# Twitch配信ガチャ集計システム（Discord通知対応）

 たぬえさの「当選時に外部プログラム実行」をトリガーに、当選データを集計しDiscordにチャットとして結果を反映できます。

 またブラウザ画面で結果を確認でき、ステータスを変更することができます。（もちろんOBSの埋め込みにも使えます）

 動作環境：windows 64bit　（Windows 11 (x64) で確認済み）

## 導入方法
**ファイルをダウンロード**
- Githubからzipなどでダウンロードして解凍してください
- 以下のディレクトリにあるzipがシステムになりますので、任意の場所に解凍・保存してください
```
app\gacha-windows-x64.zip
```
<br>

**OBSに呼び出しプログラムを設定**
- 以下のプログラムが自動起動するように設定
```
保存した場所\gacha-windows-x64\scripts\serve_api.bat
```
できそうになければめんどうですが手動で立ち上げてください。  
上記のファイルをダブルクリックで起動できます。

- たぬえさで起動するプログラムを設定
```
保存した場所\gacha-windows-x64\gacha.exe
```

- 引数を設定
```
当たりの場合：["%name%","0"]
大当たりの場合：["%name%","1"]
```
<br>

**Discord通知設定**
- `.env.local`ファイルの以下の項目にDiscordのチャンネルのWebhookリンクを貼る

例
```
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/12345/qwerty
```
Webhookリンクはチャンネルの右にある「⚙」をクリックして連携サービスからリンクを作成してください（「お名前」は任意）


## OBS 表示
以下のファイルを OBS の「ブラウザソース」に設定してください
```
public/index.html
``` 

## 使い方
たぬえさのガチャが当たると自動で集計に蓄積され、Discordチャンネルのチャットに反映されます。

※データはローカルに保存されます
<br>
<br>

集計結果を別ウィンドウで見たいときは`public/index.html`を直接ブラウザで開けば確認できます。

※OBSを使用せず集計データを更新する場合は`scripts\serve_api.bat`を手動で起動してください。（見るだけなら不要です）

## 集計画面の使い方
- ヘッダー上の「表示」プルダウンは集計データのバージョンです。  
 「＋新規作成」を押すと今の集計がバックアップされ、新たなバージョンを作成します。  
 過去の集計を見たいときはその時の「日時」を選択してください。  
 過去のデータを今のバージョンとして復元する場合は「復元」ボタンを押してください。  
<br>
<br>

- 「状態」を変更するとDiscordにも反映されます
- 「参考画像」も変更すると反映されます

## Discord通知のカスタマイズ
`setting.json`で以下の項目から内容をカスタマイズできます。

- タイトル: `discordTitle`（既定: 集計（最新））
- フィールド見出し:
  - `discordHeaderGif`（既定: `---大当たり（Gif）---`）
  - `discordHeaderIllustration`（既定: `---当たり（イラスト）---`）
- ステータス絵文字:
  - `discordEmojiDone`（既定: ✅）
  - `discordEmojiProgress`（既定: 🎨）
  - `discordEmojiNone`（既定: ⏳）
 - 参考画像ラベル:
   - `discordRefLabelYes`（既定: ●）
   - `discordRefLabelNo`（既定: ○）
<br>
<br>
<br>

# 開発者向け

## ディレクトリ
- `data/current.json`（現在値）、`data/data.js`（表示用）
- `logs/`（イベントJSON, app.log）、`backups/`（リセット時バックアップ）

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
  - 行の表示形式: `ステータス絵文字 [参考画像あり/なし] ユーザー名`（例: `✅ [参考画像あり] userA` / `🎨 [参考画像なし] userB`）。
  - 並び順: `[Gif]` グループ→`[Ilst]` グループ、各グループは名前昇順（グループ見出しは `discordHeaderGif` / `discordHeaderIllustration`）。
  - メッセージIDのマッピングは `data/discord_map.json` に保存（キーは `__SUMMARY__`）。
  
### ON/OFF 切替
- 設定ファイルの `discordEnabled`（true/false）で切り替え可能。
- 互換のため、環境変数 `DISCORD_NOTIFY` が truthy（1/true/yes/on）の場合も有効になります。
- Webhook URL が未設定の場合は通知処理をスキップし、エラーにはなりません（`app.log` に info を出力）。

### セッションと旧メッセージ
- 新規作成（リセット）時にセッションIDを発行し、Discordの集約メッセージもセッションごとに別メッセージで管理します（`discordNewMessagePerSession`）。
- 旧メッセージはアーカイブ扱いとして、先頭に見出しを付与します（`discordArchiveOldSummary`）。
  - 見出し例: `[アーカイブ 2025/09/13 12:34]`（`discordArchiveLabel` と現在日時から自動生成、日本語表記）


 
## UI の絵文字設定（状態プルダウン）
- 画面の状態プルダウンは `setting.json` の `discordEmojiNone/Progress/Done` に準拠します。
- API `GET /api/settings` から読み込み、反映します。

## ショートカット（バッチ）
- `scripts\reset.bat` リセット（バックアップ→初期化）
- `scripts\serve_api.bat [port]` ローカルAPIサーバー起動（既定: 3010）
- `scripts\restore.bat BACKUP_NAME` バックアップから復元（例: `scripts\restore.bat 2025-09-13_120000.json`）

## 配布ZIP作成
- PowerShellで作成: `scripts\package_dist.ps1`（バージョン付は `-WithVersion`）
- BATラッパ: `scripts\package_dist.bat`
- 出力先: プロジェクト直下の `app/` フォルダ
  - 例: `app/gacha-windows-x64.zip`（`-WithVersion`時は `app/gacha-<ver>-windows-x64.zip`）
- 同梱物: `gacha.exe`, `README.md`, `public/`, `scripts/`（運用に必要なもののみ）, `data/`, `logs/`, `backups/`
- 初期ファイル: `data/current.json`（空状態を投入済み）
- 設定ファイル: ルートの `setting.json` をそのまま同梱（現行値が配布デフォルトになります）。
 - 環境変数ファイル: `.env.local` を同梱（内容は `.env.example` をコピー）
