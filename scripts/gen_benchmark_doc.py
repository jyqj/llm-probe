#!/usr/bin/env python3
"""Generate benchmark summary doc from all model+effort JSON results."""
import json, os, sys, glob

def load_results(benchmark_dir):
    """Load all benchmark result files, return list of (model, effort, report)."""
    results = []
    for path in sorted(glob.glob(os.path.join(benchmark_dir, "*.json"))):
        fname = os.path.basename(path).replace(".json", "")
        parts = fname.rsplit("_", 1)
        if len(parts) != 2:
            continue
        model, effort = parts
        with open(path) as f:
            report = json.load(f)
        results.append((model, effort, report))
    return results

def gen(benchmark_dir, out_path):
    results = load_results(benchmark_dir)
    if not results:
        print("No results found")
        return

    lines = []
    a = lines.append

    a("# Claude Console — Intelligence Benchmark 基线")
    a("")
    a("> SWE-Atlas-QnA 124 题，按模型 × 思考力度 (effort) 全量测试记录。")
    a("> 渠道: Claude Console (`https://api.anthropic.com`)")
    a("")

    # Overview table
    a("## 总览")
    a("")
    a("| 模型 | Effort | 得分 | 通过率 | 完成 | 错误 | 耗时 |")
    a("|---|---|---|---|---|---|---|")
    for model, effort, r in results:
        score = f"{r['score_total']:.1f}" if r.get('score_total') is not None else "—"
        pr = f"{r['pass_rate']:.1f}%" if r.get('pass_rate') is not None else "—"
        tc = r.get('task_completed', 0)
        tt = r.get('task_total', 0)
        te = r.get('task_errors', 0)
        elapsed = r.get('elapsed_ms', 0)
        elapsed_str = f"{elapsed//1000}s" if elapsed < 600000 else f"{elapsed//60000}m"
        a(f"| {model} | {effort} | {score} | {pr} | {tc}/{tt} | {te} | {elapsed_str} |")
    a("")

    # Per-model effort comparison
    models = {}
    for model, effort, r in results:
        models.setdefault(model, []).append((effort, r))

    a("## 模型 × Effort 对比")
    a("")

    for model, efforts in models.items():
        a(f"### {model}")
        a("")
        a("| Effort | 得分 | 通过率 | 评估数 | 通过数 | 错误数 | 耗时 |")
        a("|---|---|---|---|---|---|---|")
        for effort, r in efforts:
            score = f"{r['score_total']:.1f}" if r.get('score_total') is not None else "—"
            pr = f"{r['pass_rate']:.1f}%" if r.get('pass_rate') is not None else "—"
            evl = r.get('total_evaluated', 0)
            psd = r.get('total_passed', 0)
            te = r.get('task_errors', 0)
            elapsed = r.get('elapsed_ms', 0)
            elapsed_str = f"{elapsed//1000}s" if elapsed < 600000 else f"{elapsed//60000}m"
            a(f"| {effort} | {score} | {pr} | {evl} | {psd} | {te} | {elapsed_str} |")
        a("")

    # Per-question scores across all combos
    a("## 逐题得分矩阵")
    a("")

    # Collect all task IDs in order from first result
    first_results = results[0][2].get('results', [])
    task_ids = [r['task']['task_id'] for r in first_results]
    task_cats = {r['task']['task_id']: r['task'].get('category', '') for r in first_results}
    task_langs = {r['task']['task_id']: r['task'].get('language', '') for r in first_results}

    # Build header
    combo_keys = [(m, e) for m, e, _ in results]
    header_labels = [f"{m.replace('claude-','').replace('-','')[:8]}_{e[:3]}" for m, e in combo_keys]

    a(f"| # | Task ID | Lang | Category | {' | '.join(header_labels)} |")
    a(f"|---|---|---|---|{'|'.join(['---'] * len(combo_keys))}|")

    for qi, tid in enumerate(task_ids):
        cat = task_cats.get(tid, '')[:25]
        lang = task_langs.get(tid, '')
        scores = []
        for _, _, r in results:
            task_result = None
            for tr in r.get('results', []):
                if tr['task']['task_id'] == tid:
                    task_result = tr
                    break
            if task_result is None:
                scores.append("—")
            elif task_result.get('error'):
                scores.append("ERR")
            elif task_result.get('score') is not None:
                s = task_result['score']
                if s == 1.0:
                    scores.append("1.0")
                elif s == 0.0:
                    scores.append("0.0")
                else:
                    scores.append(f"{s:.2f}")
            else:
                scores.append("—")
        a(f"| {qi+1} | `{tid[:12]}` | {lang} | {cat} | {' | '.join(scores)} |")
    a("")

    # Error summary
    has_errors = False
    for _, _, r in results:
        if r.get('task_errors', 0) > 0:
            has_errors = True
            break
    if has_errors:
        a("## 错误统计")
        a("")
        a("| 模型 | Effort | 超时 | 过载 (529) | 其他 |")
        a("|---|---|---|---|---|")
        for model, effort, r in results:
            if r.get('task_errors', 0) == 0:
                continue
            timeout = sum(1 for res in r.get('results', []) if 'deadline exceeded' in res.get('error', ''))
            overload = sum(1 for res in r.get('results', []) if '529' in res.get('error', ''))
            other = r['task_errors'] - timeout - overload
            a(f"| {model} | {effort} | {timeout} | {overload} | {other} |")
        a("")
        a("> 超时：runner 内置 3 分钟/题限制，对复杂 SWE 题 + thinking 可能不足。")
        a("")

    a("---")
    a("")
    a(f"*共 {len(results)} 组运行，{len(task_ids)} 道题*")

    with open(out_path, 'w') as f:
        f.write('\n'.join(lines) + '\n')
    print(f"Written {out_path} ({len(lines)} lines)")

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print(f"usage: {sys.argv[0]} <benchmark-dir> <output.md>")
        sys.exit(1)
    gen(sys.argv[1], sys.argv[2])
