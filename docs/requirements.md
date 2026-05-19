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
5. Git リポジトリ判定および remote URL 取得を必要とするサブコマンドは `resolve` / `switch` / `exec` / `print-env` とすること。ただし `switch` および `exec -- gh ...` は Git リポジトリ外でも unconditional な `default: true` rule を利用できること。
6. `check` は設定ファイル全体と `gh` 認証状態を検査するサブコマンドであり、Git リポジトリ外でも実行できること。
7. `init` は設定ファイル生成のみを行うサブコマンドであり、Git リポジトリ外でも実行できること。
8. `install` は shell hook 設定のみを行うサブコマンドであり、Git リポジトリ外でも実行できること。
9. `--remote <name>` は `resolve` / `switch` / `exec` / `print-env` に適用できること。`check` / `init` / `install` では指定不可とし、指定された場合は一般エラーとして扱うこと。

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
8. host は DNS hostname または IDNA hostname として妥当な値のみ許可し、空文字、制御文字、空白、shell metacharacter、path separator を含む host は解釈不能として扱うこと。
9. port 付き URL は初版では非対応とし、`https://host:port/owner/repo.git` および `ssh://git@host:port/owner/repo.git` は解釈不能として扱うこと。
10. HTTPS URL の query / fragment は初版では非対応とし、存在する場合は解釈不能として扱うこと。
11. scheme および host の比較は case-insensitive とし、scheme は小文字へ正規化すること。
12. owner および repository 名は空文字、`.`、`..`、制御文字、空白、path separator を含んではならないこと。
13. URL path の percent-encoding は初版では decode せず、`%2F` など path separator を表す encoded 値を含む URL は解釈不能として扱うこと。
14. path 末尾の `/`、空 path segment、IPv6 literal、SCP-like SSH の port 指定、`repository.git.git` は初版では解釈不能として扱うこと。
15. IDNA hostname は ASCII punycode へ正規化した値を host として扱うこと。

### 7.2.1 正規化 remote URL

ルール照合および出力で利用する正規化 remote URL は、解析結果から以下の形式で生成すること。

1. HTTPS remote は `https://<host>/<owner>/<repo>.git` として正規化すること。
2. SSH remote は `ssh://git@<host>/<owner>/<repo>.git` として正規化すること。
3. host は小文字化し、repo 末尾は必ず `.git` 付きに統一すること。
4. owner および repo の大小文字は remote URL の値を保持すること。
5. userinfo は正規化 remote URL には含めないこと。
6. password や token を含む元 URL は内部保持してよいが、ログ、標準出力、標準エラー、JSON、verbose 出力には出力しないこと。
7. 人間向けに元 URL を表示する必要がある場合は、userinfo の password component を `***` に置換し、username component も token らしい値の場合は `***` に置換すること。
8. `url_user` が token らしい値の場合、内部の rule 照合では元の値を利用してよいが、標準出力、標準エラー、JSON、verbose 出力では `***` として出力すること。

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
   5. `host + default: true` 一致
4. 複数 rule が同じ優先順位で一致した場合、設定ファイルで先に定義された rule を採用すること。
5. `remote_url` glob の照合対象は、秘匿情報を含む元 URL ではなく、7.2.1 で定義した正規化 remote URL とすること。
6. 一致 rule がない場合の動作を `noop` または `error` で設定できること。
7. `on_unmatched=noop` の場合、`resolve` / `switch` / `print-env` は正常終了し、account と rule を未解決として出力すること。
8. `exec` は `on_unmatched` の値にかかわらず、rule 未一致の場合は既定で子プロセスを実行せず rule 未一致エラーとして扱うこと。
9. `exec --allow-unmatched -- <cmd...>` が明示された場合のみ、rule 未一致でも認証確認および切替を行わず、`GH_HOST` を付与して指定コマンドを実行してよいこと。ただし shell hook 用の `--allow-unmatched-if-noop` が指定され、かつ `on_unmatched=noop` の場合も同様に実行してよいこと。
10. `owner` glob の照合対象は owner のみとし、host や repo は含めないこと。
11. glob は Go の `path.Match` 相当の文法を採用し、`*`, `?`, `[]` をサポートすること。`**` は特別扱いしないこと。
12. glob 照合は case-sensitive とすること。ただし host の一致のみ小文字正規化後に比較すること。
13. rule の `host` は完全一致のみとし、glob は許可しないこと。
14. rule 内に `url_user` / `remote_url` / `owner` を複数指定した場合、同一優先順位の候補としてではなく AND 条件として扱うこと。
15. rule の優先順位判定では、一致した rule のうち最も高い一致種別をその rule の順位とすること。ただし、指定済みの他条件が不一致の rule は採用してはならないこと。
16. `default: true` が指定された rule は、同一 host で他の rule が一致しない場合の最下位フォールバックとして採用すること。
17. unconditional な `default: true` rule とは、`default: true` が指定され、かつ `url_user` / `remote_url` / `owner` を指定しない rule を指すこと。Git リポジトリ外の fallback では、設定ファイルで最初に定義された unconditional な `default: true` rule を採用し、その rule の `host` を対象 host として利用すること。

### 7.4 `gh` 認証状態確認

1. 切替前に対象 host に対して `gh auth status` 相当の確認を行えること。[cite:6]
2. 対象 host に未ログインである場合は切替を実施せず、エラーとして扱うこと。[cite:6][cite:25]
3. 設定上要求される account が `gh` 側で切替対象として存在しない場合はエラーとすること。[cite:25]
4. `gh` コマンドが見つからない場合は一般エラーとして扱うこと。
5. 認証確認は非対話で実行し、対象 host は `gh auth status --hostname <host>` により明示すること。
6. `gh auth status --json` を使用する場合、終了コードだけで認証済みと判断してはならない。JSON の account / active / token 状態を解析し、対象 host の usable な認証が存在することを確認すること。
7. `GH_TOKEN`, `GITHUB_TOKEN`, `GH_ENTERPRISE_TOKEN`, `GITHUB_ENTERPRISE_TOKEN` のいずれかが環境変数に存在する場合、初版では `switch` / `exec` / `check` を認証エラーとして fail closed すること。ただし `resolve` と `print-env` は警告なしで実行してよい。
8. `gh` が対話プロンプトを要求する状況はエラーとして扱い、標準入力から回答を待たないこと。
9. token 環境変数チェックは `switch` / `exec` / `check` の開始時に、rule 未一致判定や `on_unmatched=noop` より先に実施すること。
10. `gh auth status --json` を利用する場合は、対象 host に対して active account と切替候補 account の一覧を取得できることを前提とすること。
11. active account は対象 host 上で現在 active と明示された account とし、該当 account が存在しない、token 状態が不明、JSON フィールドが欠落または未知形式の場合は fail closed で認証エラーとして扱うこと。
12. 要求 account が切替候補一覧に存在しない場合は `account_unavailable` とすること。
13. `gh` subprocess には `GH_HOST=<host>` を明示的に設定し、token 系環境変数は検出時点で fail して subprocess へ渡さないこと。`GH_CONFIG_DIR` は親プロセスの値を継承してよいが、README にその影響を明記すること。

### 7.5 アカウント切替

1. 現在の active account が期待 account と異なる場合のみ切替を実施すること。
2. 切替には `gh auth switch` を利用すること。[cite:25]
3. 同一 account がすでに active の場合は何も変更せず成功終了すること。
4. `gh` 呼び出し時は対象 host を `GH_HOST` 環境変数で明示し、かつ `gh auth switch --hostname <host> --user <account>` により host と account を明示すること。[cite:4]
5. `ghautoswitch switch` は親 shell の環境変数を変更しないこと。shell へ `GH_HOST` を反映したい場合は `print-env` を使用すること。
6. `switch` および `exec` は OS user の実効 `GH_CONFIG_DIR` 認証ストア単位の排他 lock を取得してから認証確認、必要な切替、切替後確認を行うこと。
7. 排他 lock の既定パスは `${XDG_RUNTIME_DIR}/ghautoswitch/<lock-key>.lock` とし、`XDG_RUNTIME_DIR` が未設定の場合は OS の一時ディレクトリ配下のユーザー専用ディレクトリを利用すること。
8. lock 取得待ちの既定 timeout は 10 秒とし、timeout 時は一般エラーとして終了すること。
9. `<lock-key>` は実効 `GH_CONFIG_DIR` の正規化パスから生成したファイル名安全な文字列を利用すること。`GH_CONFIG_DIR` が未設定の場合は `gh` の既定設定ディレクトリ相当のパスを利用すること。
10. lock 親ディレクトリは permission `0700` とし、group writable / world writable、symlink、所有者が現在ユーザー以外の場合は一般エラーとして扱うこと。
11. lock は OS の advisory file lock または同等の原子的な仕組みで実装し、プロセス異常終了時に stale lock file が残っても次回実行を恒久的に妨げないこと。

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
| `install` | bash / zsh / fish に `gh` 自動切替 hook を設定する |

### 7.7 `exec` の動作

1. `exec -- <cmd...>` は `switch` と同じ解決・認証確認・切替処理を行った後、指定されたコマンドを子プロセスとして実行すること。
2. 子プロセスには少なくとも `GH_HOST=<resolved host>` を付与すること。
3. 子プロセスの標準入力、標準出力、標準エラーは原則として呼び出し元へ透過すること。
4. 子プロセスが起動した場合、`ghautoswitch exec` の終了コードは子プロセスの終了コードに従うこと。
5. 切替前処理でエラーになった場合、子プロセスは実行せず `ghautoswitch` 自身の終了コードを返すこと。
6. rule 未一致の場合は、`--allow-unmatched` が指定されていない限り子プロセスを実行しないこと。ただし shell hook 用の `--allow-unmatched-if-noop` が指定され、かつ `on_unmatched=noop` の場合は子プロセスを実行してよいこと。
7. 子プロセス起動直前に active account を再確認し、期待 account と異なる場合は子プロセスを起動せず認証エラーとして終了すること。
8. 子プロセス実行中は既定で OS user の実効 `GH_CONFIG_DIR` 認証ストア単位の lock を保持し、子プロセス終了後に解放すること。
9. `exec --` 以降のコマンドが空の場合は一般エラーとして扱うこと。
10. 子プロセスは shell を介さず、`cmd[0]` を実行ファイル、`cmd[1:]` を引数配列としてそのまま渡して起動すること。
11. 子プロセスの起動自体に失敗した場合は一般エラーとして扱い、子プロセスの終了コードとは区別すること。
12. `--json` 指定時でも、子プロセス起動後は `ghautoswitch` 独自の JSON を標準出力へ出力しないこと。前処理エラー時のみ 7.11 の JSON エラー形式で出力すること。
13. `--allow-unmatched` 指定時、および `--allow-unmatched-if-noop` と `on_unmatched=noop` の組み合わせで rule 未一致を許可する場合は、認証確認および切替を行わず、`GH_HOST` は remote URL から解決した host を付与して子プロセスを実行すること。
14. Git リポジトリ外で `exec -- gh ...` を実行した場合、`--allow-unmatched` の有無にかかわらず、設定ファイルで最初に定義された unconditional な `default: true` rule があれば、その rule の `host` と `account` を使って切替後に子プロセスを実行すること。該当 rule がない場合は `not_git_repository` として失敗すること。

### 7.8 `print-env` の動作

1. `print-env` は POSIX shell の `eval` で利用できる `export KEY=VALUE` 形式を標準出力へ出力すること。
2. 初版で出力する環境変数は `GH_HOST` とすること。
3. `print-env` は `gh auth switch` を実行しないこと。
4. rule 未一致でも remote URL から host を解決できた場合は `GH_HOST` を出力すること。
5. 出力値は POSIX shell の single quote で必ず quote し、single quote を含む値は `'\''` 形式で escape すること。
6. `GH_HOST` に出力できる値は 7.2 で妥当と判定された host のみとすること。
7. 出力例は `export GH_HOST='github.com'` とすること。

### 7.9 `check` の動作

1. `check` は設定ファイル全体を検査対象とすること。
2. 設定ファイルの構文、必須項目、未知キー、glob の妥当性を検査すること。
3. rule に登場する host ごとに `gh` 認証状態を確認すること。
4. rule に登場する account が、対象 host の `gh` 切替候補として存在するか確認すること。
5. 複数の問題がある場合は、可能な範囲でまとめて報告すること。
6. `check` は Git リポジトリを必要とせず、`--remote` を受け付けないこと。
7. token 環境変数が存在する場合、設定ファイルの構文検査までは実施してよいが、`gh` 認証状態検査は実施せず、最終結果は認証エラーとして扱うこと。

### 7.10 `init` の動作

1. `init` は設定ファイルの親ディレクトリが存在しない場合に作成すること。
2. 既定 config path を使う場合、親ディレクトリは permission `0700`、設定ファイルは permission `0600` で作成すること。
3. `GHAUTOSWITCH_CONFIG` により config path が上書きされている場合も、作成する設定ファイルは permission `0600` とすること。
4. 設定ファイルがすでに存在する場合、既定では上書きせず設定エラーとして終了すること。
5. 上書きは `init --force` が指定された場合のみ許可すること。
6. 生成する雛形は `version`, `defaults`, 空でない `rules` 例、コメントによる最小説明を含むこと。
7. `init --print` が指定された場合はファイルを書き込まず、雛形を標準出力へ出力すること。

### 7.10.1 `install` の動作

1. `install` は bash / zsh / fish の shell 設定ファイルに `gh` 関数 hook を追加または更新できること。
2. `--shell bash|zsh|fish` により対象 shell を明示できること。未指定の場合は `SHELL` 環境変数から判定すること。
3. 既定の書き込み先は bash が `~/.bashrc`、zsh が `~/.zshrc`、fish が `~/.config/fish/config.fish` とすること。
4. hook は `gh-auto-switch exec --allow-unmatched-if-noop -- gh ...` を実行し、切替から元の `gh` コマンド終了まで排他 lock を保持すること。
5. `gh-auto-switch exec` の切替前処理が失敗した場合、元の `gh` コマンドは実行しないこと。
6. hook は管理コメントブロックで囲み、再実行時は既存ブロックを置換して重複追加しないこと。
7. `install --print` が指定された場合はファイルを書き込まず、hook を標準出力へ出力すること。

### 7.11 出力要件

1. 人間向け出力は標準出力に要点を簡潔に表示すること。
2. 自動化連携用に JSON 出力をサポートすること。
3. エラー時は非ゼロ終了コードを返すこと。
4. ログには認証情報、password、token を出力しないこと。
5. `--json` の正常系出力は、少なくとも `host`, `owner`, `repo`, `url_user`, `account`, `rule`, `matched`, `action` を含む JSON object とすること。
6. `--json` のエラー出力は、少なくとも `error.code`, `error.message` を含む JSON object とすること。
7. 未解決値は JSON では `null` として表現すること。空文字列で未解決を表現してはならない。
8. `matched` は boolean とすること。
9. `rule` は一致した rule の `name` 文字列、未一致時は `null` とし、rule object は出力しないこと。
10. `action` は `none`, `resolved`, `switched`, `exec`, `unmatched_noop`, `check_passed`, `init_created`, `install_created` のいずれかとすること。
11. エラー時の標準出力には JSON 以外を混在させないこと。人間向けエラーは標準エラーへ出力すること。
12. `url_user` は出力用の値とし、token らしい値の場合は `***`、存在しない場合は `null` とすること。
13. 内部照合に利用した生の `url_user` は JSON、verbose、人間向け出力に含めてはならないこと。
14. サブコマンドごとの正常系 `action` は以下を基本とすること。
   - `resolve`: `resolved` または `unmatched_noop`
   - `switch`: `switched` または `none` または `unmatched_noop`
   - `exec`: 子プロセス起動前の独自 JSON は出力しない。前処理のみを表す必要がある場合は `exec`
   - `print-env`: `resolved` または `unmatched_noop`
   - `check`: `check_passed`
   - `init`: `init_created`。`init --print` は `none`
   - `install`: `install_created`。`install --print` は `none`

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
  on_unmatched: error
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

  - name: github-default
    host: github.com
    default: true
    account: hogehoge
```

### 8.3 設定バリデーション

1. `version` は必須とする。
2. `rules[].name` はユニークであること。
3. `rules[].account` は必須とする。
4. `host` 未指定 rule は初版では禁止する。
5. 不正な glob や不正な YAML は起動時にエラーとする。
6. `defaults.on_unmatched` は `noop` または `error` のみ許可する。
7. `defaults.on_unauthenticated` は初版では `error` のみ許可する。
8. 未知のトップレベルキー、未知の `defaults` キー、および未知の rule キーは設定エラーとする。
9. `version` は整数 `1` のみ許可すること。
10. `rules` は 1 件以上の配列とすること。
11. `rules[].host` は 7.2 の host 妥当性条件を満たすこと。
12. `rules[].account` は空文字、空白、制御文字を含んではならないこと。
13. `rules[].url_user` は空文字、空白、制御文字を含んではならないこと。
14. `rules[].remote_url` と `rules[].owner` は同一 rule 内で併用してよい。ただし指定された条件はすべて一致必須とし、優先順位は 7.3 の一致種別に従うこと。
15. 設定ファイルまたは親ディレクトリが group writable / world writable の場合、既定では設定エラーとして扱うこと。
16. 設定ファイルが symlink の場合、初版では設定エラーとして扱うこと。
17. `defaults.remote` は省略可能とし、省略時は `origin` とすること。指定する場合は空文字、空白、制御文字を含んではならないこと。
18. `rules[].name` は空文字、空白、制御文字を含んではならないこと。
19. `rules[].remote_url` は 7.2.1 の正規化 remote URL に対する glob として評価可能な文字列であること。
20. `rules[].owner` は owner に対する glob として評価可能な文字列であること。
21. `GHAUTOSWITCH_CONFIG` に相対パスが指定された場合は、カレントディレクトリ基準の絶対パスへ変換して扱うこと。
22. `GHAUTOSWITCH_CONFIG` 指定時も、設定ファイルおよび親ディレクトリの owner / permission / symlink 検査を実施すること。

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
- userinfo の username component が 40 文字以上、または `ghp_`, `github_pat_`, `gho_`, `ghu_`, `ghs_`, `ghr_` で始まる場合は token らしい値として扱い、出力上は `***` にマスクすること。
- `print-env` の出力は shell injection の影響を受けないよう、host validation と shell quote を必須とすること。
- `exec` は子プロセス終了まで OS user の実効 `GH_CONFIG_DIR` 認証ストア単位の lock を保持すること。ただし `gh` の active account は本ツールを介さない外部プロセスにより変更され得る制約を README に明記すること。
- `exec` は shell を介して子プロセスを起動してはならないこと。
- `gh` subprocess へ渡す環境変数は原則として親プロセスから継承してよいが、`GH_HOST` は対象 host で上書きし、token 系環境変数は検出時点で fail closed すること。

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
| Git リポジトリではない | 終了コード 1 でエラー終了。ただし設定ファイルで最初に定義された unconditional な `default: true` rule がある場合、`switch` はその account へ切替して成功し、`exec -- gh ...` はその account へ切替後に子プロセスを実行する |
| 指定 remote が存在しない | 終了コード 1 でエラー終了 |
| remote URL を解釈できない | 終了コード 1 でエラー終了 |
| rule 未一致かつ `on_unmatched=error` | 終了コード 4 でエラー終了 |
| rule 未一致かつ `on_unmatched=noop` | `resolve` / `switch` / `print-env` は正常終了。ただし account / rule は未解決として扱う |
| `exec` で rule 未一致かつ `--allow-unmatched` なし | 終了コード 4 でエラー終了。ただし `--allow-unmatched-if-noop` と `on_unmatched=noop` の組み合わせは除く |
| 設定ファイルが存在しない | 終了コード 2 でエラー終了。ただし `init` は除く |
| 設定ファイルが不正 | 終了コード 2 でエラー終了 |
| `init` で設定ファイルが既に存在し、`--force` なし | 終了コード 2 でエラー終了 |
| 設定ファイルまたは親ディレクトリの permission / owner / symlink が不正 | 終了コード 2 でエラー終了 |
| token 環境変数が存在する状態で `switch` / `exec` / `check` を実行 | 終了コード 3 でエラー終了 |
| 対象 host に未ログイン | 終了コード 3 でエラー終了[cite:6] |
| 対象 account が `gh` に存在しない | 終了コード 3 でエラー終了[cite:25] |
| `gh auth switch` 失敗 | 終了コード 3 でエラー終了[cite:25] |
| `gh` コマンドが見つからない | 終了コード 1 でエラー終了 |
| 排他 lock の取得 timeout | 終了コード 1 でエラー終了 |
| `exec --` 以降のコマンドが空 | 終了コード 1 でエラー終了 |
| `exec` の子プロセス起動に失敗 | 終了コード 1 でエラー終了 |
| サブコマンドに対応しないオプションが指定された | 終了コード 1 でエラー終了 |
| `exec` の子プロセスが失敗 | 子プロセスの終了コードを返す |

`--json` 指定時の `error.code` は以下のいずれかとする。

- `not_git_repository`
- `remote_not_found`
- `remote_url_unparseable`
- `config_not_found`
- `config_invalid`
- `unmatched_rule`
- `token_env_present`
- `host_unauthenticated`
- `account_unavailable`
- `gh_not_found`
- `gh_switch_failed`
- `lock_timeout`
- `exec_command_empty`
- `exec_spawn_failed`
- `invalid_arguments`
- `internal_error`

`error.code` と終了コードの対応は以下を基本とする。

| `error.code` | 終了コード |
|---|---:|
| `not_git_repository` | 1 |
| `remote_not_found` | 1 |
| `remote_url_unparseable` | 1 |
| `config_not_found` | 2 |
| `config_invalid` | 2 |
| `unmatched_rule` | 4 |
| `token_env_present` | 3 |
| `host_unauthenticated` | 3 |
| `account_unavailable` | 3 |
| `gh_not_found` | 1 |
| `gh_switch_failed` | 3 |
| `lock_timeout` | 1 |
| `exec_command_empty` | 1 |
| `exec_spawn_failed` | 1 |
| `invalid_arguments` | 1 |
| `internal_error` | 1 |

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
- `4`: rule 未一致（`on_unmatched=error`、または `exec` で rule 未一致かつ許可オプションなしの場合）

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
10. `on_unmatched=noop` の場合、`resolve` / `switch` / `print-env` は rule 未一致でも終了コード 0 で account / rule 未解決として扱われること。
11. `exec` は rule 未一致時、`--allow-unmatched` がない限り子プロセスを実行せず終了コード 4 で失敗すること。ただし `--allow-unmatched-if-noop` と `on_unmatched=noop` の組み合わせ、および Git リポジトリ外の `exec -- gh ...` で unconditional な `default: true` rule を使う場合は子プロセスを実行できること。
12. `exec` は切替後に shell を介さず子プロセスを実行し、子プロセスの終了コードを返すこと。
13. `exec` は子プロセス終了まで OS user の実効 `GH_CONFIG_DIR` 認証ストア単位の lock を保持すること。
14. `remote_url` glob が 7.2.1 の正規化 remote URL に対して照合されること。
15. `print-env` が `export GH_HOST='github.com'` 形式で shell-safe な出力を行うこと。
16. `GH_TOKEN` 等の token 環境変数が存在する場合、`switch` / `exec` / `check` が終了コード 3 で失敗すること。
17. `gh auth switch` 呼び出しが `--hostname` と `--user` を明示し、対話プロンプトに依存しないこと。
18. `init` が既存設定を既定では上書きせず、作成時に `0600` の設定ファイルを生成すること。
19. 設定ファイルの未知キー、symlink、group/world writable permission を設定エラーとして検出できること。
20. `--json` の未解決値が `null` で出力され、エラー時に `error.code` と `error.message` を含むこと。
21. token らしい `url_user` が JSON、verbose、人間向け出力で `***` としてマスクされること。
22. `check` が Git リポジトリ外でも実行でき、`--remote` 指定を一般エラーとして扱うこと。

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
