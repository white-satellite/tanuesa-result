 # テスト仕様書：たぬえさ外部トリガー集計ツール

 ## 1. 対象
 - 実行ファイル: `gacha`（Windows: `gacha.exe`）
 - 主機能: 当選更新、data.js生成、バックアップ/リセット、ログ/履歴

 ## 2. テスト環境
 - OS: Windows 11 x64（本番想定）/ 開発用にLinuxでも可
 - 依存: なし（単体バイナリ）
 - 初期状態: 空の作業ディレクトリ（`data/`, `logs/`, `backups/`なし）

## 3. テストケース
 1) 正常系: 当たり更新
 - 手順: `gacha "userA" 0` 実行
 - 期待: `current.json` に `userA.hit=1`、`jackpot=0`、`flags.illust=true`、`flags.gif=false`、`data.js` 生成、`logs/*.json` 1件以上、`app.log`にupdate記録

 2) 正常系: 大当たり更新
 - 手順: `gacha "userA" 1` 実行
 - 期待: `userA.jackpot=1`、`flags.gif=true`

 3) 正常系: 複数ユーザーとソート表示（目視）
 - 手順: `gacha "userB" 0` を3回、`gacha "ユーザーＣ" 0` を1回
 - 期待: `userB.hit=3`、`ユーザーＣ.hit=1`、`public/index.html` + `data.js` で表が描画される（当たり降順）

 4) 境界: 日本語・空白含む名前
 - 手順: `gacha "山田 太郎" 0`
 - 期待: `current.json` に正しく記録

 5) エラー: 無効なフラグ
 - 手順: `gacha "userX" 2`（終了コード≠0）
 - 期待: `current.json` 変更なし、エラーログ記録

 6) リセット/バックアップ
 - 手順: `gacha reset`
 - 期待: `backups/*.json` が作成、`current.json` が空初期化、`data.js` も空相当へ更新

7) 再生成
- 手順: `gacha gen-datajs`
- 期待: 直近の `current.json` 内容で `data.js` 再生成

8) 20ユーザー×1回（混在フラグ）
- 手順: `test/manual/test_invoke_20_users.ps1 -Reset -Prefix "USER" -Count 20`
- 期待: `ユーザー01`〜`ユーザー20` が存在。5の倍数（`05,10,15,20`）は `jackpot=1, hit=0, gif=true`、他は `hit=1, jackpot=0, gif=false`（全員 `illust` は当たりユーザーのみ true）。

 ## 4. 判定基準
 - すべての期待結果を満たすこと
 - 異常時に非0終了し、既存データを破壊しないこと

 ## 5. 補足
 - 連続同一呼び出しの重複抑止は将来拡張。現段階では機能しなくても良い（仕様外）。
