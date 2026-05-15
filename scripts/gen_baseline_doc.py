#!/usr/bin/env python3
"""Generate baseline markdown doc from probe result JSON."""
import json, sys, os

def gen(json_path, channel_name, out_path):
    with open(json_path) as f:
        r = json.load(f)

    lines = []
    a = lines.append

    a(f"# {channel_name} — {r['model']} 基线")
    a("")
    a(f"> 渠道 `{channel_name}` 使用 `{r['target']}` 运行全部 probe 的真实记录。")
    a("")

    a("## 测试元数据")
    a("")
    a("| 字段 | 值 |")
    a("|---|---|")
    a(f"| 渠道 | {channel_name} |")
    a(f"| 目标 | `{r['target']}` |")
    a(f"| 模型 | `{r['model']}` |")
    a(f"| 时间 | {r['timestamp'][:19]} |")
    a(f"| 总耗时 | {r['elapsed_ms']}ms ({r['elapsed_ms']//1000}s) |")
    b = r.get('billing', {})
    a(f"| 预估消耗 | input {b.get('input_tokens',0)} / output {b.get('output_tokens',0)} tokens, ${b.get('total_cost',0)} |")
    s = r.get('score', {})
    a(f"| 评分 | **{s.get('grade','')} {s.get('total_score',0)}/100** |")
    total_checks = sum(len(pr.get('checks') or []) for pr in r.get('probe_results', []))
    passed_checks = sum(1 for pr in r.get('probe_results', []) for c in (pr.get('checks') or []) if c['pass'])
    a(f"| Checks | {passed_checks}/{total_checks} passed |")
    sk = r.get('skipped_probes', [])
    if sk:
        a(f"| 跳过 Probes | {', '.join(sk)} |")
    a("")

    a("## 分类得分")
    a("")
    a("| 分类 | 得分 | 通过率 |")
    a("|---|---|---|")
    for cat in s.get('categories', []):
        a(f"| {cat['label']} | {cat['score']}/{cat['weight']} | {cat['passed']}/{cat['total']} |")
    a("")

    a("## 逐 Probe 结果")
    a("")

    for pr in r.get('probe_results', []):
        checks = pr.get('checks') or []
        total = len(checks)
        passed = sum(1 for c in checks if c['pass'])
        status = "PASS" if passed == total else f"PARTIAL {passed}/{total}"
        a(f"### {pr['probe_id']} — {pr['label']} [{status}] ({pr['latency_ms']}ms)")
        a("")
        a("| Check | 状态 | 实际值 | 详情 |")
        a("|---|---|---|---|")
        for c in checks:
            icon = "PASS" if c['pass'] else "FAIL"
            actual = c.get('actual', '').replace('|', '\\|')
            detail = c.get('detail', '').replace('|', '\\|')
            if len(actual) > 80:
                actual = actual[:77] + "..."
            if len(detail) > 120:
                detail = detail[:117] + "..."
            a(f"| `{c['name']}` | {icon} | {actual} | {detail} |")
        a("")

    failed = [c for c in r.get('checks', []) if not c['pass']]
    if failed:
        a("## 失败 Checks")
        a("")
        a("| Check | 说明 |")
        a("|---|---|")
        for c in failed:
            a(f"| `{c['name']}` | {c.get('detail', '')} |")
        a("")

    a("---")
    a("")
    a(f"*生成时间: {r['timestamp'][:19]}*")

    with open(out_path, 'w') as f:
        f.write('\n'.join(lines) + '\n')
    print(f"Written {out_path} ({len(lines)} lines)")

if __name__ == '__main__':
    if len(sys.argv) < 4:
        print(f"usage: {sys.argv[0]} <json> <channel-name> <output.md>")
        sys.exit(1)
    gen(sys.argv[1], sys.argv[2], sys.argv[3])
