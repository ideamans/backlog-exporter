# backlog-tasks

Backlog から未完了タスク（課題）を抽出し、ファイルに出力するCLIツールです。

## 機能

- 指定したプロジェクトの未完了タスクを一括取得
- 親子課題の階層構造を保持した出力
- 3つの出力フォーマット（TXT, Markdown, JSON）に対応
- 担当者でのフィルタリング

## インストール

### GitHub Releases からダウンロード

[Releases](https://github.com/miyanaga/backlog-exporter/releases) ページから、お使いのOS・アーキテクチャに合ったバイナリをダウンロードしてください。

### Go でインストール

```bash
go install github.com/miyanaga/backlog-exporter/cmd/backlog-tasks@latest
```

### ソースからビルド

```bash
git clone https://github.com/miyanaga/backlog-exporter.git
cd backlog-exporter
go build -o backlog-tasks ./cmd/backlog-tasks
```

## 使い方

### 基本的な使い方

```bash
# 環境変数でAPIキーを設定
export BACKLOG_API_KEY="your-api-key"

# プロジェクトキーで指定
backlog-tasks -s mycompany -p MYPROJ

# Markdown形式で出力
backlog-tasks -s mycompany -p MYPROJ -f markdown

# 出力先ディレクトリを指定
backlog-tasks -s mycompany -p MYPROJ -o ./output
```

### APIキーの取得方法

1. Backlog にログイン
2. 右上のプロフィールアイコンをクリック → 「個人設定」
3. 「API」タブを選択
4. 「新しいAPIキーを発行」をクリック

### コマンドオプション

| オプション | 短縮形 | 必須 | デフォルト | 説明 |
|-----------|--------|------|------------|------|
| `--api-key` | `-k` | △※ | - | Backlog APIキー |
| `--space` | `-s` | ○ | - | BacklogスペースID（例: `mycompany`） |
| `--domain` | `-d` | - | `backlog.com` | ドメイン |
| `--project` | `-p` | ○ | - | プロジェクトIDまたはキー |
| `--output` | `-o` | - | `./` | 出力先ディレクトリ |
| `--format` | `-f` | - | `txt` | 出力フォーマット |
| `--assignee` | `-a` | - | - | 担当者でフィルタ（ユーザーID） |
| `--help` | `-h` | - | - | ヘルプを表示 |
| `--version` | `-v` | - | - | バージョンを表示 |

※ 環境変数 `BACKLOG_API_KEY` が設定されていれば省略可

### 環境変数

コマンドラインオプションの代わりに環境変数でも設定できます。

| 変数名 | 説明 |
|--------|------|
| `BACKLOG_API_KEY` | Backlog APIキー |
| `BACKLOG_SPACE` | スペースID |
| `BACKLOG_DOMAIN` | ドメイン |

### 出力フォーマット

#### TXT形式（デフォルト）

```
================================================================================
プロジェクト: MYPROJ - マイプロジェクト
取得日時: 2024-11-27 14:30:52
未完了タスク数: 15件（親課題: 8件、子課題: 7件）
================================================================================

[MYPROJ-100] 大きな機能開発
  状態: 処理中
  優先度: 高
  担当者: yamada
  期限日: 2024-12-01

  ├─ [MYPROJ-101] サブタスク1: 設計
  │    状態: 処理済み
  │    優先度: 高
  │    担当者: yamada
  │
  └─ [MYPROJ-102] サブタスク2: 実装
       状態: 処理中
       優先度: 高
       担当者: suzuki
```

#### Markdown形式 (`-f markdown`)

見出しやテーブルを使った読みやすい形式で出力します。GitHub や Notion などでそのまま表示できます。

#### JSON形式 (`-f json`)

プログラムで処理しやすい構造化された形式で出力します。親子課題の関係は `children` フィールドで表現されます。

### 出力ファイル名

ファイル名は以下の形式で自動生成されます：

```
{プロジェクトキー}_tasks_{YYYYMMDD_HHMMSS}.{拡張子}
```

例: `MYPROJ_tasks_20241127_143052.md`

### 使用例

```bash
# 基本的な使い方（TXT形式）
backlog-tasks -s mycompany -p MYPROJ

# Markdown形式で出力
backlog-tasks -s mycompany -p MYPROJ -f markdown -o ./reports

# JSON形式で出力
backlog-tasks -s mycompany -p MYPROJ -f json

# backlog.jp ドメインを使用
backlog-tasks -s mycompany -d backlog.jp -p MYPROJ

# 特定の担当者のタスクのみ抽出
backlog-tasks -s mycompany -p MYPROJ -a 12345

# プロジェクトIDで指定
backlog-tasks -s mycompany -p 12345
```

### 実行例

```
$ backlog-tasks -s mycompany -p MYPROJ -f markdown

Connecting to mycompany.backlog.com...
Project: MYPROJ (マイプロジェクト)
Fetching statuses... done
Fetching issues... 100
Fetching issues... 200
Fetching issues... 234/234 (complete)
Building hierarchy... done

Summary:
  Total issues: 234
  Parent issues: 156
  Child issues: 78

Output: ./MYPROJ_tasks_20241127_143052.md
Done!
```

## 未完了タスクの定義

以下のステータスを「未完了」として扱います：

- 未対応
- 処理中
- 処理済み

「完了」ステータスの課題は出力されません。

## 対応ドメイン

- `backlog.com`（デフォルト）
- `backlog.jp`
- `backlogtool.com`

## エラーについて

| エラー | 原因と対処法 |
|--------|-------------|
| `API key is required` | APIキーが設定されていません。`--api-key` または環境変数 `BACKLOG_API_KEY` を設定してください |
| `Authentication failed` | APIキーが無効です。正しいAPIキーか確認してください |
| `Project not found` | 指定したプロジェクトが見つかりません。プロジェクトキーまたはIDを確認してください |
| `Cannot write to directory` | 出力先ディレクトリに書き込めません。ディレクトリが存在し、書き込み権限があるか確認してください |

## ライセンス

MIT License
