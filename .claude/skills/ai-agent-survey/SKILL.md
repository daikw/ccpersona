---
name: ai-agent-survey
description: "AI コーディングエージェントツールの仕様を網羅的に調査する。フック・イベント・設定・統合ポイントを対象ツールごとに調べ、比較表を生成する。キーワード: AIエージェント調査, フック仕様, agent tools, hook survey"
user-invocable: true
argument-hint: "<調査観点 or 対象ツール名> (省略時: 全主要ツールのフック・イベント仕様)"
allowed-tools:
  - WebSearch
  - WebFetch
  - Task
  - Write
  - Read
  - Glob
  - Grep
  - AskUserQuestion
---

# AI Agent Survey - AIエージェントツール仕様調査

AI コーディングエージェントツールの仕様（特にフック・イベント・統合ポイント）を網羅的に調査し、比較表を生成する。

## When to Use

- 新しい AI エージェントツールへの対応を検討するとき
- 既存の対応ツール一覧が古くなったとき（定期調査）
- 特定のフック（session start, stop, notify など）の対応状況を横断確認したいとき
- ccpersona などのツールに新プラットフォームを追加するか判断するとき

## 調査対象ツール（デフォルト）

以下を網羅する。`$ARGUMENTS` で対象を絞れる。

| カテゴリ | ツール |
|----------|--------|
| **CLI 型** | Claude Code, OpenAI Codex CLI, Aider, Amp (Sourcegraph) |
| **IDE 拡張** | Cursor, Windsurf, Cline, Roo Code, Continue.dev, GitHub Copilot |
| **その他** | 調査時点で注目度の高いもの |

## 調査観点

各ツールについて以下を調べる:

1. **フック/イベントの有無** - hooks, events, callbacks が存在するか
2. **session-start 相当** - セッション/会話の開始を検出できるか
3. **session-stop 相当** - 終了タイミングを検出できるか
4. **turn/notify 相当** - AI 応答ごとのイベントがあるか
5. **設定方法** - JSON, TOML, env var, config file のどれか
6. **渡されるデータ** - JSON stdin / 環境変数 / 引数 など
7. **公式ドキュメントURL**
8. **最終確認日**

## 実行手順

### Phase 0: 引数の解析

`$ARGUMENTS` を確認し:
- 空の場合: 全ツールを調査（デフォルト）
- ツール名指定: そのツールのみ詳細調査
- 観点指定（例: "session-start のみ"）: 絞って調査

### Phase 1: 並列調査

対象ツールを **Task エージェントで並列調査** する。各エージェントへの指示:

```
<ツール名> の AI エージェントとしてのフック・イベント仕様を調査してください。
WebSearch と WebFetch を使い公式ドキュメント・GitHub・リリースノートを確認すること。
以下の項目を調べて返してください:
- フック/イベント機能の有無
- session-start/stop/turn 相当のイベント名と設定方法
- フックに渡されるデータの形式
- 公式ドキュメントURL
- 情報の鮮度（いつのドキュメント/バージョンか）
確認や質問は不要です。調査結果のみ返してください。
```

**典型的な分担例（全調査時）:**
1. Agent A: Claude Code + Codex CLI
2. Agent B: Cursor + Windsurf
3. Agent C: Cline + Roo Code
4. Agent D: Aider + Continue.dev + GitHub Copilot
5. Agent E: 最新注目ツール（Amp など）

### Phase 2: 統合・比較表の生成

調査結果をまとめ、以下のフォーマットで出力する。

## 出力フォーマット

```markdown
# AI エージェントツール フック仕様調査

_調査日: {date}_
_対象バージョン: 各ツールの最新安定版_

---

## サマリー比較表

| ツール | フック機能 | session-start | session-stop | turn/notify | 設定方法 |
|--------|-----------|--------------|-------------|------------|---------|
| Claude Code | ✅ | ✅ SessionStart | ✅ Stop | ✅ Notification | JSON hooks |
| Cursor | ✅ | ✅ sessionStart | ✅ stop | ✅ afterAgentResponse | JSON hooks.json |
| Codex CLI | 部分的 | ❌ | ❌ | ✅ agent-turn-complete | TOML notify |
| ... | | | | | |

## ツール別詳細

### {ツール名}

- **フック機能**: あり/なし/部分的
- **session-start**: {イベント名 or ❌} - {設定方法}
- **session-stop**: {イベント名 or ❌}
- **turn/notify**: {イベント名 or ❌}
- **渡されるデータ**: {形式・フィールド}
- **設定ファイル**: {パス・フォーマット}
- **公式ドキュメント**: {URL}
- **備考**: {注意点・制限など}

...（各ツール繰り返し）

## ccpersona 対応状況

| ツール | 対応済み | 対応フック | 未対応・検討中 |
|--------|---------|-----------|--------------|
| Claude Code | ✅ | SessionStart, Stop, Notification | - |
| Cursor | ✅ | sessionStart, afterAgentResponse, stop | - |
| Codex CLI | ✅ | agent-turn-complete | session-start なし |
| ... | ❌ | - | {対応可能なら実装方針} |

## 新規対応候補

調査で判明した未対応ツールのうち、対応価値があるものと実装方針を記載。

## Open Questions / 要確認事項

- 未確認の仕様や、ドキュメントが見つからなかった項目
```

## 注意事項

- **情報の鮮度を必ず記載**: AI ツールは仕様変更が頻繁
- **公式一次情報を優先**: GitHub リポジトリ, 公式ドキュメントを確認
- **「なさそう」は「ない」ではない**: 見つからなくても「未確認」と書く
- ccpersona の実装状況との対比を必ず含める
