# ghautoswitch 要件定義書

## 1. 文書概要

本書は、Git リポジトリの remote URL に応じて GitHub CLI（`gh`）の利用アカウントを自動切替する CLI ツール `ghautoswitch` の要件を定義するものである。[cite:4][cite:19][cite:25]

`gh` はホスト単位で認証状態とアクティブアカウントを持ち、`gh auth status` により状態確認、`gh auth switch` によりアカウント切替が可能であるため、`ghautoswitch` はその判定と実行を補助するラッパー CLI として設計する。[cite:6][cite:25]

## 2. 背景

GitHub CLI は GitHub.com と GitHub Enterprise Server をまたいだ利用、および同一ホスト内の複数アカウント運用をサポートしている。[cite:4][cite:25]

一方でローカル環境では、作業対象リポジトリごとに利用すべき GitHub アカウントが異なる場合があり、手動で `gh auth switch` を実行する運用は煩雑で切替漏れを生みやすい。[cite:25]

また Git の remote URL は HTTPS 形式と SSH 形式で設定でき、リポジトリの接続先情報は `git remote` に保存されるため、remote URL はアカウント切替判定の主要な入力情報として利用できる。[cite:17][cite:19]

## 3. 目的

`ghautoswitch` の目的は以下のとおりとする。

- カレントディレクトリ配下の Git リポジトリから remote URL を取得し、対象ホストおよび利用すべき `gh` アカウントを決定する。
- 必要に応じて `gh auth switch` を実行し、`gh` のアクティブアカウントを期待値へ揃える。[cite:25]
- GitHub.com と GitHub Enterprise Server の両方に対応する。[cite:4]
- 設定をユーザーディレクトリ配下にグローバル保存し、複数リポジトリで共通利用できるようにする。
- HTTPS remote URL に含まれる userinfo username（例: `https://hogehoge@github.com/org/repo.git` の `hogehoge`）を、アカウント識別ヒントとして利用できるようにする。[cite:19]

## 4. 対象範囲

### 4.1 対象

- ローカル Git リポジトリを対象とする CLI ツール
- GitHub CLI (`gh`) がインストール済みである環境
- GitHub.com および GitHub Enterprise Server
- macOS / Linux を主対象とし、Windows は将来対応とする

### 4.2 対象外

- `gh` 自体の認証処理の代替実装
- Git remote の自動書換え
- Git 操作そのものの代行
- GUI アプリケーションの提供
- 複数 VCS（GitLab, Bitbucket など）への初版対応

## 5. 用語定義

| 用語 | 定義 |
|---|---|
| host | remote URL から抽出される GitHub ホスト名。例: `github.com`、`github.example.com`。[cite:4][cite:19] |
| account | 対象 host 上で `gh` が認識する GitHub ログイン名。`gh auth switch` に渡す切替対象ユーザー名であり、初版では `personal` / `work` のようなツール独自の別名解決は提供しない。[cite:25] |
| remote | Git リポジトリに設定された接続先。既定は `origin` とする。[cite:17][cite:19] |
| url_user | HTTPS remote URL の userinfo username。例: `https://hogehoge@github.com/...` の `hogehoge`。[cite:19] |
| rule | remote URL から account を解決するための設定単位 |

## 6. 想定利用シナリオ

### 6.1 GitHub.com の個人・業務アカウント切替

- 個人用リポジトリでは `github.com/personal-org/*` に対し個人用 GitHub ログイン名を使う。
- 会社用リポジトリでは `github.com/company-org/*` に対し業務用 GitHub ログイン名を使う。
- `ghautoswitch switch` 実行時に remote URL から自動判定して切替を行う。

### 6.2 GitHub Enterprise Server との切替

- `github.example.com/team/*` では Enterprise 用アカウントへ切り替える。
- 必要に応じて `GH_HOST` を設定し、`gh` の既定ターゲットを対象ホストに合わせる。[cite:4]

### 6.3 HTTPS userinfo による識別

- remote URL が `https://hogehoge@github.com/org/repo.git` の場合、`url_user=hogehoge` を最優先の識別ヒントとして用いる。
- `url_user` に対応する rule が存在する場合は、その rule を優先採用する。
- ただし最終的な切替可否は、対象 host 上で当該ユーザーが `gh` により切替可能であることを前提とする。[cite:6][cite:25]

## 7. 機能要件

### 7.1 リポジトリ判定

1. カレントディレクトリが Git リポジトリか判定できること。
2. 既定 remote 名は `origin` とすること。
3. オプションにより remote 名を上書きできること。
4. remote URL を取得できること。[cite:17][cite:19]

### 7.2 remote URL 解析

1. 以下の形式を解釈できること。[cite:19]
   - `https://github.com/owner/repo.git`
   - `https://github.com/owner/repo`
   - `https://hogehoge@github.com/owner/repo.git`
   - `git@github.com:owner/repo.git`
   - `git@github.com:owner/repo`
   - `ssh://git@github.com/owner/repo.git`
   - `https://github.example.com/owner/repo.git`
   - `git@github.example.com:owner/repo.git`
2. 解析結果として少なくとも以下を抽出できること。
   - scheme
   - host
   - owner または org
   - repository 名
   - url_user（存在する場合のみ）
3. SSH URL の `git@` は固定ユーザーとして扱い、アカウント識別には使わないこと。[cite:19]
4. repository 名の末尾 `.git` は任意とし、抽出結果の repository 名では `.git` を除去すること。
5. host は小文字へ正規化して比較すること。
6. 初版では GitHub の一般的な `owner/repo` 形式を対象とし、path が 2 階層未満または 3 階層以上の remote URL は解釈不能として扱うこと。
7. `https://user:password@host/...` のように password を含む場合、内部解析は許可するがログ出力では必ずマスクすること。

### 7.3 ルールマッチング

1. グローバル設定ファイルに定義された rule に基づき account を解決できること。
2. 初版のマッチ条件は以下をサポートすること。
   - `host`
   - `url_user`
   - `remote_url` パターン（glob）
   - `owner` パターン（glob）
3. rule の優先順位は以下とすること。
   1. `host + url_user` 一致
   2. `host + remote_url` 一致
   3. `host + owner` 一致
   4. `host` のみ一致
4. 複数 rule が同じ優先順位で一致した場合、設定ファイルで先に定義された rule を採用すること。
5. `remote_url` glob の照合対象は、秘匿情報を含む元 URL ではなく、credential をマスクした正規化済み remote URL とすること。
6. 一致 rule がない場合の動作を `noop` または `error` で設定できること。
7. `on_unmatched=noop` の場合、`resolve` / `switch` / `print-env` は正常終了し、account と rule を未解決として出力すること。`exec` は切替せずに指定コマンドを実行すること。

### 7.4 `gh` 認証状態確認

1. 切替前に対象 host に対して `gh auth status` 相当の確認を行えること。[cite:6]
2. 対象 host に未ログインである場合は切替を実施せず、エラーとして扱うこと。[cite:6][cite:25]
3. 設定上要求される account が `gh` 側で切替対象として存在しない場合はエラーとすること。[cite:25]
4. `gh` コマンドが見つからない場合は一般エラーとして扱うこと。

### 7.5 アカウント切替

1. 現在の active account が期待 account と異なる場合のみ切替を実施すること。
2. 切替には `gh auth switch` を利用すること。[cite:25]
3. 同一 account がすでに active の場合は何も変更せず成功終了すること。
4. `gh` 呼び出し時は対象 host を `GH_HOST` 環境変数で明示すること。[cite:4]
5. `ghautoswitch switch` は親 shell の環境変数を変更しないこと。shell へ `GH_HOST` を反映したい場合は `print-env` を使用すること。

### 7.6 実行モード

以下のサブコマンドを提供すること。

| サブコマンド | 機能 |
|---|---|
| `resolve` | 現在の remote から host / owner / repo / url_user / account / rule を表示する |
| `switch` | 必要な場合のみ `gh` のアクティブアカウントを切り替える |
| `exec -- <cmd...>` | 必要な場合のみ切替を行い、対象 host を示す環境変数を付与して任意コマンドを実行する |
| `print-env` | shell 連携用に `GH_HOST` 等の環境変数を出力する |
| `check` | 設定ファイルと `gh` 認証状態の整合性を検査する |
| `init` | 設定ファイルの雛形を生成する |

### 7.7 `exec` の動作

1. `exec -- <cmd...>` は `switch` と同じ解決・認証確認・切替処理を行った後、指定されたコマンドを子プロセスとして実行すること。
2. 子プロセスには少なくとも `GH_HOST=<resolved host>` を付与すること。
3. 子プロセスの標準入力、標準出力、標準エラーは原則として呼び出し元へ透過すること。
4. 子プロセスが起動した場合、`ghautoswitch exec` の終了コードは子プロセスの終了コードに従うこと。
5. 切替前処理でエラーになった場合、子プロセスは実行せず `ghautoswitch` 自身の終了コードを返すこと。
6. `on_unmatched=noop` の場合は切替せず、`GH_HOST` は remote URL から解決した host を付与して子プロセスを実行すること。

### 7.8 `print-env` の動作

1. `print-env` は POSIX shell の `eval` で利用できる `export KEY=VALUE` 形式を標準出力へ出力すること。
2. 初版で出力する環境変数は `GH_HOST` とすること。
3. `print-env` は `gh auth switch` を実行しないこと。
4. rule 未一致でも remote URL から host を解決できた場合は `GH_HOST` を出力すること。

### 7.9 `check` の動作

1. `check` は設定ファイル全体を検査対象とすること。
2. 設定ファイルの構文、必須項目、未知キー、glob の妥当性を検査すること。
3. rule に登場する host ごとに `gh` 認証状態を確認すること。
4. rule に登場する account が、対象 host の `gh` 切替候補として存在するか確認すること。
5. 複数の問題がある場合は、可能な範囲でまとめて報告すること。

### 7.10 出力要件

1. 人間向け出力は標準出力に要点を簡潔に表示すること。
2. 自動化連携用に JSON 出力をサポートすること。
3. エラー時は非ゼロ終了コードを返すこと。
4. ログには認証情報、password、token を出力しないこと。
5. `--json` の正常系出力は、少なくとも `host`, `owner`, `repo`, `url_user`, `account`, `rule`, `matched`, `action` を含む JSON object とすること。
6. `--json` のエラー出力は、少なくとも `error.code`, `error.message` を含む JSON object とすること。

## 8. 設定要件

### 8.1 設定ファイル保存先

- 既定パスは `~/.config/ghautoswitch/config.yml` とする。
- 環境変数 `GHAUTOSWITCH_CONFIG` により上書き可能とする。
- 設定はユーザーディレクトリ配下のグローバル設定として扱う。

### 8.2 設定ファイル形式

YAML を採用する。初版の想定形式は以下とする。

```yaml
version: 1

defaults:
  remote: origin
  on_unmatched: noop
  on_unauthenticated: error

rules:
  - name: github-personal-by-url-user
    host: github.com
    url_user: hogehoge
    account: hogehoge

  - name: github-work-by-owner
    host: github.com
    owner: my-company
    account: work-user

  - name: github-work-by-url
    host: github.com
    remote_url: "https://github.com/my-company/*"
    account: work-user

  - name: ghe-work
    host: github.example.com
    account: enterprise-user
```

### 8.3 設定バリデーション

1. `version` は必須とする。
2. `rules[].name` はユニークであること。
3. `rules[].account` は必須とする。
4. `host` 未指定 rule は初版では禁止する。
5. 不正な glob や不正な YAML は起動時にエラーとする。
6. `defaults.on_unmatched` は `noop` または `error` のみ許可する。
7. `defaults.on_unauthenticated` は初版では `error` のみ許可する。
8. 未知のトップレベルキーおよび未知の rule キーは設定エラーとする。

## 9. 非機能要件

### 9.1 可用性・保守性

- 単一バイナリで配布できること。
- 外部依存は `git` と `gh` に限定すること。
- 判定ロジックと `gh` 実行ロジックを分離し、単体テスト可能にすること。

### 9.2 セキュリティ

- token や password をログ、標準出力、エラー出力へ表示しないこと。
- `url_user` は識別ヒントとしてのみ利用し、資格情報としては扱わないこと。[cite:19]
- `gh` 側の認証状態を信頼源とし、切替可否は `gh auth status` / `gh auth switch` の結果で判断すること。[cite:6][cite:25]
- remote URL に password が含まれる場合、出力上は `https://user:***@host/owner/repo.git` の形式で password のみをマスクすること。
- userinfo の password component は credential として扱い、`--verbose` および `--json` を含む全出力でマスクすること。userinfo の username component は `url_user` として扱ってよい。

### 9.3 パフォーマンス

- 通常の `resolve` / `switch` 実行は対話的利用を阻害しない時間で完了すること。
- 追加のネットワークアクセスは極力行わず、基本的には `git` と `gh` のローカル状態参照で完結すること。

### 9.4 可観測性

- `--verbose` で判定過程を出力できること。
- `--json` で機械可読な詳細結果を出力できること。
- 秘匿情報は verbose でもマスクすること。

## 10. エラー要件

| 条件 | 動作 |
|---|---|
| Git リポジトリではない | エラー終了 |
| 指定 remote が存在しない | エラー終了 |
| remote URL を解釈できない | エラー終了 |
| rule 未一致かつ `on_unmatched=error` | 終了コード 4 でエラー終了 |
| rule 未一致かつ `on_unmatched=noop` | 正常終了。ただし account / rule は未解決として扱う |
| 対象 host に未ログイン | エラー終了[cite:6] |
| 対象 account が `gh` に存在しない | エラー終了[cite:25] |
| `gh auth switch` 失敗 | エラー終了[cite:25] |
| `exec` の子プロセスが失敗 | 子プロセスの終了コードを返す |

## 11. インターフェース要件

### 11.1 CLI 例

```bash
ghautoswitch resolve
ghautoswitch switch
ghautoswitch switch --remote upstream
ghautoswitch exec -- gh pr list
eval "$(ghautoswitch print-env)"
ghautoswitch check --json
```

### 11.2 終了コード

- `0`: 正常終了
- `1`: 一般エラー
- `2`: 設定エラー
- `3`: 認証エラー
- `4`: rule 未一致（`on_unmatched=error` の場合のみ）

## 12. 受入基準

以下を満たした場合に初版リリース可能と判定する。

1. `origin` の HTTPS / SSH remote から host, owner, repo を正しく抽出できること。[cite:19]
2. `https://hogehoge@github.com/org/repo.git` から `url_user=hogehoge` を正しく抽出できること。[cite:19]
3. `url_user` 一致 rule が owner 一致 rule より優先されること。
4. `gh auth status` で未認証 host を検出できること。[cite:6]
5. 期待 account と active account が異なる場合にのみ `gh auth switch` を実行すること。[cite:25]
6. 期待 account がすでに active の場合は無変更で成功すること。[cite:25]
7. `print-env` で `GH_HOST` を適切に出力できること。[cite:4]
8. 設定ファイルがユーザーディレクトリ配下のグローバル設定として読み込まれること。
9. 秘匿情報がログに出力されないこと。
10. `on_unmatched=noop` の場合、rule 未一致でも終了コード 0 で account / rule 未解決として扱われること。
11. `exec` は切替後に子プロセスを実行し、子プロセスの終了コードを返すこと。

## 13. 実装方針

実装言語は Go を第一候補とする。単一バイナリ配布、YAML 取扱い、`git` / `gh` の subprocess 制御、クロスプラットフォーム対応の観点で適しているためである。

モジュール構成は以下を推奨する。

- `config`: 設定ロード、バリデーション
- `gitremote`: Git リポジトリ判定、remote URL 取得
- `parser`: HTTPS / SSH remote 解析、`url_user` 抽出
- `matcher`: rule 優先順位判定
- `ghadapter`: `gh auth status` / `gh auth switch` 実行
- `cmd`: CLI サブコマンド実装

## 14. 将来拡張

- 複数 remote を横断した自動選択
- `include` を用いた設定分割
- Windows ネイティブ対応強化
- シェルフック連携（`cd` 時の自動判定）
- GitLab / Bitbucket など他ホストへの拡張

## 15. 補足方針

本ツールは remote URL に含まれる情報から `gh` アカウントを推定する補助ツールであり、認証そのものの正本管理は `gh` が担う。[cite:6][cite:25]

特に HTTPS userinfo username は「どのアカウントを使いたいか」を示すヒントとしては有用であるが、実際の資格情報そのものではないため、判定と認証を分離して扱う設計を採用する。[cite:19][cite:25]
