# Backlog 未完了タスク抽出CLI - 仕様書 v2

## 概要

Backlog APIを使用して、指定したプロジェクトから未完了のタスク（課題）を取得し、テキストファイルに保存するCLIツール。

---

## 基本情報

| 項目 | 内容 |
|------|------|
| 言語 | Go |
| コマンド名 | `backlog-tasks`（仮） |

---

## 認証

Backlog APIへの認証にはAPIキーを使用します。

### APIキーの指定方法（優先順位順）

1. コマンドライン引数: `--api-key` または `-k`
2. 環境変数: `BACKLOG_API_KEY`

---

## コマンドライン引数

| 引数 | 短縮形 | 必須 | デフォルト | 説明 |
|------|--------|------|------------|------|
| `--api-key` | `-k` | △※ | - | Backlog APIキー |
| `--space` | `-s` | ○ | - | BacklogスペースID（例: `mycompany`） |
| `--domain` | `-d` | × | `backlog.com` | ドメイン（`backlog.com`, `backlog.jp`, `backlogtool.com`） |
| `--project` | `-p` | ○ | - | プロジェクトIDまたはプロジェクトキー |
| `--output` | `-o` | × | `./` | 出力先ディレクトリ |
| `--format` | `-f` | × | `txt` | 出力フォーマット（`txt`, `json`, `markdown`） |
| `--assignee` | `-a` | × | - | 担当者でフィルタ（ユーザーID） |
| `--help` | `-h` | × | - | ヘルプを表示 |
| `--version` | `-v` | × | - | バージョンを表示 |

※ 環境変数 `BACKLOG_API_KEY` が設定されていれば省略可

---

## 環境変数

| 変数名 | 説明 |
|--------|------|
| `BACKLOG_API_KEY` | Backlog APIキー |
| `BACKLOG_SPACE` | スペースID（オプション） |
| `BACKLOG_DOMAIN` | ドメイン（オプション） |

---

## 未完了タスクの定義

Backlogのデフォルト状態において、以下を「未完了」とみなします：

- `未対応`（status.id = 1）
- `処理中`（status.id = 2）
- `処理済み`（status.id = 3）

※ `完了`（status.id = 4）以外を取得

### 補足
プロジェクトによってカスタム状態がある場合は、`/api/v2/projects/:projectIdOrKey/statuses` で状態一覧を取得し、「完了」以外のものを抽出するロジックが必要。

---

## プロジェクト指定

プロジェクトは **ID（数値）** と **キー（文字列）** の両方に対応：
```bash
# プロジェクトキーで指定
backlog-tasks -s mycompany -p MYPROJ

# プロジェクトIDで指定
backlog-tasks -s mycompany -p 12345
```

### 判定ロジック
```
入力値が数値のみ → プロジェクトIDとして扱う
入力値に英字を含む → プロジェクトキーとして扱う
```

---

## ページネーション

### 自動全件取得

Backlog APIは1回のリクエストで最大100件まで取得可能。100件を超える場合は自動的にページネーションを処理し、全件を取得する。

### 実装方針
```
1. count=100, offset=0 で最初のリクエスト
2. 100件取得できた場合 → offset += 100 で次のリクエスト
3. 100件未満になるまで繰り返し
4. すべての結果をマージ
```

### 進捗表示
```
Fetching issues... 100/100
Fetching issues... 200/234
Fetching issues... 234/234 (complete)
```

---

## 親子課題の扱い

### 基本方針

- 親課題と子課題の両方を取得
- 出力時に **階層構造** で表示
- 親を持たない課題はトップレベルに配置

### 構造化ロジック
```
1. 全課題を取得
2. parentIssueId が null → 親課題（またはスタンドアロン）
3. parentIssueId が設定されている → 子課題
4. 親課題ごとに子課題をグルーピング
5. 階層構造で出力
```

---

## 出力ファイル

### ファイル名規則
```
{project_key}_tasks_{YYYYMMDD_HHMMSS}.{format}
```

例: `MYPROJ_tasks_20241127_143052.md`

---

## 出力フォーマット

### Markdown出力（階層構造）
```markdown
# MYPROJ - マイプロジェクト 未完了タスク一覧

> 取得日時: 2024-11-27 14:30:52  
> 未完了タスク数: 15件（親課題: 8件、子課題: 7件）

---

## [MYPROJ-100] 大きな機能開発
| 項目 | 内容 |
|------|------|
| 状態 | 処理中 |
| 優先度 | 高 |
| 担当者 | yamada |
| 期限日 | 2024-12-01 |
| 作成日 | 2024-11-01 |
| 更新日 | 2024-11-25 |

### 子課題

#### [MYPROJ-101] サブタスク1: 設計
| 項目 | 内容 |
|------|------|
| 状態 | 処理済み |
| 優先度 | 高 |
| 担当者 | yamada |
| 期限日 | 2024-11-15 |

#### [MYPROJ-102] サブタスク2: 実装
| 項目 | 内容 |
|------|------|
| 状態 | 処理中 |
| 優先度 | 高 |
| 担当者 | suzuki |
| 期限日 | 2024-11-25 |

---

## [MYPROJ-200] 単独タスク（子課題なし）
| 項目 | 内容 |
|------|------|
| 状態 | 未対応 |
| 優先度 | 中 |
| 担当者 | - |
| 期限日 | - |

---
```

### TXT出力（階層構造）
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
  作成日: 2024-11-01
  更新日: 2024-11-25

  ├─ [MYPROJ-101] サブタスク1: 設計
  │    状態: 処理済み
  │    優先度: 高
  │    担当者: yamada
  │    期限日: 2024-11-15
  │
  └─ [MYPROJ-102] サブタスク2: 実装
       状態: 処理中
       優先度: 高
       担当者: suzuki
       期限日: 2024-11-25

--------------------------------------------------------------------------------

[MYPROJ-200] 単独タスク（子課題なし）
  状態: 未対応
  優先度: 中
  担当者: (未割当)
  期限日: -

--------------------------------------------------------------------------------
```

### JSON出力（階層構造）
```json
{
  "project": {
    "id": 1,
    "key": "MYPROJ",
    "name": "マイプロジェクト"
  },
  "exportedAt": "2024-11-27T14:30:52+09:00",
  "summary": {
    "total": 15,
    "parentIssues": 8,
    "childIssues": 7
  },
  "issues": [
    {
      "id": 100,
      "issueKey": "MYPROJ-100",
      "summary": "大きな機能開発",
      "status": "処理中",
      "priority": "高",
      "assignee": "yamada",
      "dueDate": "2024-12-01",
      "createdAt": "2024-11-01T10:00:00Z",
      "updatedAt": "2024-11-25T15:30:00Z",
      "children": [
        {
          "id": 101,
          "issueKey": "MYPROJ-101",
          "summary": "サブタスク1: 設計",
          "status": "処理済み",
          "priority": "高",
          "assignee": "yamada",
          "dueDate": "2024-11-15",
          "createdAt": "2024-11-02T10:00:00Z",
          "updatedAt": "2024-11-14T15:30:00Z"
        },
        {
          "id": 102,
          "issueKey": "MYPROJ-102",
          "summary": "サブタスク2: 実装",
          "status": "処理中",
          "priority": "高",
          "assignee": "suzuki",
          "dueDate": "2024-11-25",
          "createdAt": "2024-11-05T10:00:00Z",
          "updatedAt": "2024-11-20T15:30:00Z"
        }
      ]
    },
    {
      "id": 200,
      "issueKey": "MYPROJ-200",
      "summary": "単独タスク（子課題なし）",
      "status": "未対応",
      "priority": "中",
      "assignee": null,
      "dueDate": null,
      "createdAt": "2024-11-15T10:00:00Z",
      "updatedAt": "2024-11-20T15:30:00Z",
      "children": []
    }
  ]
}
```

---

## 使用するBacklog API

### 1. 課題一覧の取得
```
GET /api/v2/issues?apiKey={apiKey}
```

**パラメータ:**
- `projectId[]`: プロジェクトID
- `statusId[]`: 状態ID（未完了のもの）
- `count`: 100（最大値）
- `offset`: ページネーション用
- `sort`: `created`
- `order`: `asc`

### 2. プロジェクトの状態一覧取得
```
GET /api/v2/projects/:projectIdOrKey/statuses?apiKey={apiKey}
```

### 3. プロジェクト情報取得（キー→ID変換用）
```
GET /api/v2/projects/:projectIdOrKey?apiKey={apiKey}
```

---

## 処理フロー
```
1. 引数・環境変数を解析
2. APIキーの検証
3. プロジェクト情報を取得（ID/キー解決）
4. プロジェクトの状態一覧を取得
5. 「完了」以外の状態IDを特定
6. 課題一覧を取得（ページネーション処理）
7. 親子関係を構造化
8. 指定フォーマットで出力
9. ファイルに保存
```

---

## 使用例

### 基本的な使用方法
```bash
# 環境変数でAPIキーを設定
export BACKLOG_API_KEY="your-api-key"

# プロジェクトキーで指定（Markdown出力）
backlog-tasks -s mycompany -p MYPROJ -o ./output -f markdown

# プロジェクトIDで指定
backlog-tasks -s mycompany -p 12345 -o ./output
```

### 実行結果
```
$ backlog-tasks -s mycompany -p MYPROJ -o ./output -f markdown

Connecting to mycompany.backlog.com...
Project: MYPROJ (マイプロジェクト)
Fetching statuses... done
Fetching issues... 100/100
Fetching issues... 200/234
Fetching issues... 234/234 (complete)
Building hierarchy... done

Summary:
  Total issues: 234
  Parent issues: 156
  Child issues: 78

Output: ./output/MYPROJ_tasks_20241127_143052.md
Done!
```

---

## エラーハンドリング

| エラー種別 | 終了コード | メッセージ例 |
|-----------|-----------|-------------|
| APIキー未設定 | 1 | `Error: API key is required. Set --api-key or BACKLOG_API_KEY` |
| 認証エラー | 2 | `Error: Authentication failed. Please check your API key` |
| プロジェクト未発見 | 3 | `Error: Project 'XXX' not found` |
| ネットワークエラー | 4 | `Error: Failed to connect to Backlog API` |
| 出力ディレクトリエラー | 5 | `Error: Cannot write to directory './output'` |
| レート制限 | 6 | `Error: Rate limit exceeded. Please wait and try again` |

---

## 今後の拡張案（オプション）

- [ ] マイルストーンでフィルタ (`--milestone`)
- [ ] カテゴリでフィルタ (`--category`)
- [ ] 優先度でフィルタ (`--priority`)
- [ ] 期限切れタスクのみ抽出 (`--overdue`)
- [ ] 複数プロジェクト対応
- [ ] CSV出力対応
- [ ] 差分出力（前回との比較）
- [ ] 子課題のみ表示 (`--children-only`)
- [ ] 親課題のみ表示 (`--parents-only`)