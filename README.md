 # Twitch配信ガチャ集計システム（外部トリガー対応）

 Windows 11 (x64) 向け。たぬえさの「当選時に外部プログラム実行」をトリガーに、当選データ(JSON)を更新し、OBSブラウザソース用 `data/data.js` を生成します。

## 使い方（外部トリガー）
- 実行形式: `gacha.exe "<winnerName>" <hitFlag>`（`hitFlag`: `0=当たり`, `1=大当たり`）
- 例: `C:\\tools\\gacha.exe "ユーザーA" 0`
- 補助コマンド: `reset`（バックアップ→初期化）、`backup`、`gen-datajs`、`gen-backup-index`、`--version`
- 任意: `serve [port]`（ローカルAPI起動。UIからリセット/履歴再生成を実行可能）
 - Discord通知（任意）: 環境変数 `DISCORD_NOTIFY=1` と `DISCORD_WEBHOOK_URL` を設定すると、更新時にDiscordへ投稿/更新します。

## OBS 表示
- `public/index.html` を OBS の「ブラウザソース」でローカルファイルとして読み込み。
- 数秒ごとに `data/data.js` を再読込し最新化。
- 履歴閲覧: ヘッダーの「表示」プルダウンでバックアップを選択（`backups/index.js` を自動読込）。
- 新規追加（リセット）: プルダウン末尾「+ 新規追加」を選択すると、現在値をバックアップして初期化。
  - 事前に `gacha.exe serve` を起動しておくと、画面から実行されます。
  - サーバ未起動時は案内に従って `gacha.exe reset` を手動実行してください。

 ## ディレクトリ
 - `data/current.json`（現在値）、`data/data.js`（表示用）
 - `logs/`（イベントJSON, app.log）、`backups/`（リセット時バックアップ）

 ## たぬえさ設定例
 - 実行ファイル: `C:\\tools\\gacha.exe`
 - 引数: `"$user$" $flag$`（たぬえさ側のプレースホルダ例）
 - うまく渡らない場合は `scripts/update_result.ps1` / `.bat` を利用し、そちらを呼び出してください。

## ビルド（開発者向け）
- 前提: Go 1.21+
- 手順: `go build -o gacha.exe ./src`

## インストール不要のビルド（Windows）
- PowerShellを「実行」: `scripts/build_portable_win.ps1`（Goのzipを取得→展開→ビルド。終了後`gacha.exe`が作成）
- 例: `pwsh -File scripts\build_portable_win.ps1`（必要なら `-GoVersion 1.21.13` `-Cleanup`）

## GitHub Actions（任意）
- `.github/workflows/build-windows.yml` を用意済み。GitHubへPushするとWindows用バイナリをArtifactとして取得可能。

## テスト
- テスト仕様書: `test/specs/test_spec.md`
- 自動テスト（Linux/WSL等）: `python3 test/auto/run_tests.py`（リポジトリ直下にLinux用`gacha`が必要）
- テスト用起動プログラム（Windows）: `test/manual/`
  - 20ユーザー×1回: `test_invoke_20_users.ps1`（BAT: `test_invoke_20_users.bat`）

## バックアップ/履歴
- `gacha.exe reset` 実行時に `backups/` にスナップショットJSONを作成し、同名の `.js` と `backups/index.js` を自動生成します。
- 履歴が表示されない場合は `gacha.exe gen-backup-index` を実行して再生成してください。

## 設定（setting.json）
- 位置: プロジェクト直下の `setting.json`（初回起動/配布ZIPに同梱）
- 項目:
  - `eventJsonLog`（true/false）: 当たる度のJSONログ（logs/日時.json）を出力するか
  - `autoServe`（true/false）: gacha.exe 実行時にAPIサーバー（`serve`）を自動起動するか
  - `serverPort`（数値）: APIサーバーのポート（既定: 3010）

## APIサーバーの起動/停止
- 自動起動: `setting.json` の `autoServe`=true の場合、`gacha.exe` 実行時に自動で `serve` を起動します。
- 手動起動: `scripts\serve_api.bat [port]`
- 停止: `scripts\stop_api.bat`（`setting.json` の `serverPort` または `-Port` 引数で指定）。内部でポートのPIDを特定して停止します。

## 運用ショートカット
- 新規作成（UIの「+ 新規追加」と同等）: `scripts\new_session.bat`
  - 現在値のバックアップ→初期化→`backups/index.js` 再生成までを一括実行します。

## Discord 連携（Webhook）
- 環境変数:
  - `DISCORD_NOTIFY=1`
  - `DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/{id}/{token}`
- 仕様（まとめ表示）:
  - current.json の内容を1メッセージに集約して投稿/更新します（表示単位で一括）。
  - 行ごとの形式:
    - 大当たり: `[Gif] ユーザー名`
    - 当たり: `[Ilst] ユーザー名`
  - 並び順: `[Gif]` グループ→`[Ilst]` グループ、各グループは名前昇順。
  - メッセージIDのマッピングは `data/discord_map.json` に保存（キーは `__SUMMARY__`）。
  
### ON/OFF 切替
- 設定ファイルの `discordEnabled`（true/false）で切り替え可能。
- 互換のため、環境変数 `DISCORD_NOTIFY` が truthy（1/true/yes/on）の場合も有効になります。
- Webhook URL が未設定の場合は通知処理をスキップし、エラーにはなりません（`app.log` に info を出力）。

### セッションと旧メッセージ
- 新規作成（リセット）時にセッションIDを発行し、Discordの集約メッセージもセッションごとに別メッセージで管理します（`discordNewMessagePerSession`）。
- 旧メッセージはアーカイブ扱いとして、先頭に見出しを付与します（`discordArchiveOldSummary`）。
  - 見出し例: `[アーカイブ 2025/09/13 12:34]`（`discordArchiveLabel` と現在日時から自動生成、日本語表記）

## ショートカット（バッチ）
- `scripts\reset.bat` リセット（バックアップ→初期化）
 - `scripts\serve_api.bat [port]` ローカルAPIサーバー起動（既定: 3010）
- `scripts\restore.bat BACKUP_NAME` バックアップから復元（例: `scripts\restore.bat 2025-09-13_120000.json`）

## 配布ZIP作成
- PowerShellで作成: `scripts\package_dist.ps1`（バージョン付は `-WithVersion`）
- BATラッパ: `scripts\package_dist.bat`
- 出力: `dist/gacha-windows-x64.zip`（`-WithVersion`時は `gacha-<ver>-windows-x64.zip`）
- 同梱物: `gacha.exe`, `README.md`, `public/`, `scripts/`（運用に必要なもののみ）, `data/`, `logs/`, `backups/`
- 初期ファイル: `data/current.json`（空状態を投入済み）
