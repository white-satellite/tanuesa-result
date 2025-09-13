# Repository Guidelines

## プロジェクト構成・モジュール配置
- ルート直下には企画仕様の `twitch配信ガチャ集計システム_rfp.md` が存在します。
- 実装開始時は以下の標準構成を採用してください。
  - `src/` 本体コード（ドメイン/機能ごとにサブフォルダ）
  - `tests/` 自動テスト（`unit/`, `integration/`）
  - `scripts/` 開発・運用スクリプト（データ移行、ユーティリティ）
  - `docs/` 仕様・設計資料、画像など
  - `assets/` 静的ファイル（アイコン、CSS、サンプルデータ）
  - `data/` 一時データ（コミットしない、必要なら `.gitignore`）

## ビルド・テスト・開発コマンド
- 依存関係: 言語/ランタイムが確定後に `README` に追記します。以下は推奨例です。
  - Node.js: `npm ci` / `npm run dev`（ローカル実行）/ `npm test`
  - Python: `pip install -r requirements.txt` / `pytest -q`
  - Make: `make setup`（初期化）、`make test`、`make lint`

## コーディング規約・命名
- インデント: JS/TS は 2 スペース、Python は 4 スペース。
- スタイル: Prettier + ESLint（JS/TS）/ Ruff + Black（Python）。保存時に自動整形を有効化。
- 命名: 変数・関数は `camelCase`、クラスは `PascalCase`、Python モジュールは `snake_case.py`、公開 API は安定化後に破壊的変更を避ける。
- ファイル: 機能単位で分割、1 ファイル 300 行程度を目安に保守性を優先。

## テスト方針
- フレームワーク: Jest/Vitest（JS/TS）または Pytest（Python）。
- 配置・命名: `tests/unit/xxx.test.ts`、`tests/unit/test_xxx.py`。
- カバレッジ: 重要モジュールは 80% 以上を目標。リグレッション再発防止のテストを優先。
- 実行例: `npm test -- --watch` / `pytest -q` / `make test`。

## コミット & PR ガイドライン
- コミット: Conventional Commits を推奨（例: `feat(api): add tally endpoint`）。
- PR: 概要、動機、主な変更点、テスト方法、関連 Issue を記載。UI 変更はスクリーンショット添付。
- スコープを小さく、レビュー可能な差分に分割。CI を通過させてからレビュー依頼。

## セキュリティ・設定
- 機密情報は `.env.local` などに保存し、` .env.example` を用意。リポジトリに秘密情報をコミットしない。
- 外部トークンは最小権限で発行し、ログに出力しない。

## エージェント向け補足
- 本ガイドはリポジトリ全体に適用。下位ディレクトリに別の `AGENTS.md` がある場合はそちらを優先。
- 仕様変更は `docs/` に記録し、コード・テスト・ドキュメントを同時に更新してください。

## Windows .bat のエンコーディングと実装指針（UTF-8問題回避）
- 文字コード/改行
  - バッチは ASCII（英数字記号のみ）で記述し、日本語は使わない（メッセージも英語推奨）。
  - エディタ保存は UTF-8(BOMなし) か ANSI を推奨。BOM 付きUTF-8は先頭に不可視文字が入りエラーの原因。
  - 改行は CRLF。
- 安全な書き方（引用・引数）
  - 先頭に `@echo off` と `setlocal`。
  - 変数代入は `set "VAR=value"` 形式（末尾スペース混入防止）。
  - 引数は `%~1` のようにチルダ展開を使い、空白/日本語を安全に受け渡し。
  - スクリプトのディレクトリは `%~dp0` を使用。
  - `start` で新しいコンソールを開く場合: `start "title" "%ComSpec%" /k ""%EXE%" serve %PORT%"`（二重引用でネスト）。
  - PowerShell 呼び出し: `powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0script.ps1" ...`。
- 制御/エスケープ
  - 遅延展開は必要時のみ有効化（`setlocal enabledelayedexpansion`）。`!` を含む値は展開に注意。
  - メタ文字は `^` でエスケープ（例: `^<Ctrl+C^>` 表示など）。
- 典型パターン
  - 実行: `"%EXE%" %~1 %~2`
  - エラー判定: `if errorlevel 1 exit /b 1` / 正常: `exit /b 0`

