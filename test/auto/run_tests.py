#!/usr/bin/env python3
import json
import os
import re
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
TESTDIR = ROOT / 'dist' / 'test-run'

def run(cmd, cwd=None, expect=0):
    p = subprocess.run(cmd, cwd=cwd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    if p.returncode != expect:
        raise AssertionError(f"cmd failed: {cmd}\nrc={p.returncode}\nstdout={p.stdout}\nstderr={p.stderr}")
    return p

def read_json(p: Path):
    with p.open('r', encoding='utf-8') as f:
        return json.load(f)

def writef(p: Path, data: str):
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(data, encoding='utf-8')

def prepare():
    if TESTDIR.exists():
        shutil.rmtree(TESTDIR)
    TESTDIR.mkdir(parents=True)
    # choose binary
    bin_src = ROOT / 'gacha'
    if not bin_src.exists():
        print('Linux test binary not found: gacha', file=sys.stderr)
        sys.exit(2)
    bin_dst = TESTDIR / 'gacha'
    shutil.copy2(bin_src, bin_dst)
    os.chmod(bin_dst, 0o755)
    return bin_dst

def assert_user(state, name, hit, jackpot, illust, gif):
    users = {u['name']: u for u in state.get('users', [])}
    u = users.get(name)
    assert u is not None, f"user {name} not found"
    assert u['hit'] == hit, (u['hit'], hit)
    assert u['jackpot'] == jackpot, (u['jackpot'], jackpot)
    assert bool(u['flags']['illust']) == bool(illust)
    assert bool(u['flags']['gif']) == bool(gif)

def load_state():
    return read_json(TESTDIR / 'data' / 'current.json')

def main():
    exe = prepare()
    passed = []

    # 1) 当たり更新
    run([str(exe), 'userA', '0'], cwd=TESTDIR)
    st = load_state()
    assert_user(st, 'userA', 1, 0, True, False)
    passed.append('1: hit update')

    # 2) 大当たり更新
    run([str(exe), 'userA', '1'], cwd=TESTDIR)
    st = load_state()
    assert_user(st, 'userA', 1, 1, True, True)
    passed.append('2: jackpot update')

    # 3) 複数ユーザー
    for _ in range(3):
        run([str(exe), 'userB', '0'], cwd=TESTDIR)
    run([str(exe), 'ユーザーＣ', '0'], cwd=TESTDIR)
    st = load_state()
    assert_user(st, 'userB', 3, 0, True, True)
    assert_user(st, 'ユーザーＣ', 1, 0, True, False)
    passed.append('3: multi users & flags')

    # 4) 日本語・空白
    run([str(exe), '山田 太郎', '0'], cwd=TESTDIR)
    st = load_state()
    assert_user(st, '山田 太郎', 1, 0, True, False)
    passed.append('4: space & jp name')

    # 5) 無効フラグ
    before = st
    p = subprocess.run([str(exe), 'userX', '2'], cwd=TESTDIR)
    assert p.returncode != 0
    st = load_state()
    # 確認: userX が追加されていない
    assert not any(u['name']=='userX' for u in st.get('users', []))
    passed.append('5: invalid flag rejected')

    # 6) リセット/バックアップ
    run([str(exe), 'reset'], cwd=TESTDIR)
    st = load_state()
    assert st.get('users') == [], st
    # backup exists
    backups = list((TESTDIR/'backups').glob('*.json'))
    assert backups, 'no backups created'
    passed.append('6: reset & backup')

    # 7) 再生成
    run([str(exe), 'gen-datajs'], cwd=TESTDIR)
    datajs = (TESTDIR/'data'/'data.js').read_text(encoding='utf-8')
    assert re.search(r"window.__GACHA_DATA__\s*=\s*\{", datajs), 'data.js invalid'
    passed.append('7: gen data.js')

    # logs
    assert any(p.suffix=='.json' for p in (TESTDIR/'logs').glob('*.json')), 'event logs missing'
    assert (TESTDIR/'logs'/'app.log').exists(), 'app.log missing'

    print('PASS:', ', '.join(passed))
    return 0

if __name__ == '__main__':
    sys.exit(main())
