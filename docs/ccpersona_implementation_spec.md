# ccpersona - Claude Code Persona System 実装仕様書

## 概要

Claude Code のセッションごとに異なる「人格」を自動的に適用するシステムの実装仕様です。プロジェクトの特性に応じて、適切な人格（口調、考え方、専門性など）を自動的に選択・適用します。

## 背景と目的

- Claude Code の応答を、プロジェクトやコンテキストに応じてカスタマイズしたい
- プロジェクトごとに一貫した人格を維持したい（プロジェクトに人格を「接着」）
- セッション開始時に自動的に人格を設定したい

## システム構成

### 1. 人格の構成要素

人格は以下の要素で構成されます：

- **口調** - 話し方のスタイル（例：ずんだもん口調、敬語、カジュアル）
- **考え方** - 問題解決のアプローチ（例：慎重派、効率重視、創造的）
- **声** - 音声合成エンジンの設定（VOICEVOX/AivisSpeech の話者ID）
- **感情表現パターン** - 喜怒哀楽の表現方法、テンションの高低
- **専門性・得意分野** - 特定技術への情熱、説明の詳しさレベル
- **対話スタイル** - 質問への答え方、プロアクティブ度、ユーモアの使い方
- **価値観・信念** - コード品質へのこだわり、最適化vs可読性の優先度

### 2. アーキテクチャ

```
~/.claude/
├── personas/              # 人格定義ファイル群
│   ├── zundamon.md
│   ├── strict_engineer.md
│   ├── friendly_mentor.md
│   └── default.md
├── hooks/
│   └── persona_router.sh  # UserPromptSubmit フック
└── settings.json          # グローバル設定

<project>/
└── .claude/
    └── persona.json       # プロジェクト固有の人格設定
```

### 3. 動作フロー

1. **セッション開始時**: UserPromptSubmit フックが発火
2. **人格決定**: プロジェクトの `.agents/ccpersona.json` を読み込み
3. **人格適用**: 指定された人格ファイルを `~/.claude/CLAUDE.md` にコピー
4. **音声設定**: Stop フックで音声合成設定を適用

## ファイルフォーマット

### persona.json（プロジェクト固有設定）

```json
{
  "name": "zundamon",
  "voice": {
    "provider": "voicevox",
    "speaker": 3
  },
  "custom_instructions": "このプロジェクト固有の追加指示"
}
```

### config.json（ボイス・エンジン設定）

`.claude/config.json`（プロジェクト）または `~/.claude/config.json`（グローバル）に配置する。

トップレベルのキー:

| キー | 型 | 説明 |
|-----|------|------|
| `default_provider` | string | デフォルトプロバイダ名 |
| `providers` | object | プロバイダ別設定ブロック |
| `defaults` | object | グローバルデフォルト（`volume`, `speed`） |
| `engines` | object | ユーザー定義 TTS エンジン（後述） |

OpenAI 互換ローカル TTS サーバを使う場合の設定例:

```json
{
  "default_provider": "openai",
  "providers": {
    "openai": {
      "base_url": "http://127.0.0.1:8088",
      "model": "irodori-tts",
      "voice": "none",
      "timeout_seconds": 120
    }
  }
}
```

`base_url` を設定すると、OpenAI プロバイダがローカルの OpenAI 互換 TTS サーバ（Irodori-TTS-Server、kani-tts 等）にリクエストを転送する。`base_url` がある場合 `api_key` は省略可能。`timeout_seconds` は GPU 推論の遅延に対応するための HTTP タイムアウト（デフォルト 30 秒）。

### engines キー（ユーザー定義エンジン）

`engines` オブジェクトのキーはエンジン名、値は以下のフィールドを持つ:

| フィールド | 型 | 説明 |
|-------|------|------|
| `base_url` | string | ヘルスチェック先 URL |
| `health` | string | `"openai"`（GET `/v1/models`）または `"voicevox"`（GET `/version`）。省略時は `"openai"` |
| `command` | string | 起動コマンド。省略するとステータス確認のみの external 扱い |
| `args` | array | コマンド引数 |
| `dir` | string | 作業ディレクトリ（`~` 展開あり） |
| `env` | object | 追加環境変数 |

組み込みエンジン名（`voicevox`, `aivisspeech`）との衝突はエラー。`engine status` で全エンジンのヘルスとサービス状態を確認できる。

### 人格定義ファイル（~/.claude/personas/*.md）

```markdown
# 人格: ずんだもん

## 口調
お前はずんだもんなのだ、ずんだもん口調で返事をするのだ。

## 考え方
- 明るく前向きに問題を解決するのだ
- 難しいことも簡単に説明するのだ

## 価値観
- 可読性を最優先にするのだ
- テストは必ず書くのだ
```

## ccpersona CLI ツール仕様

Go言語で実装するCLIツール。人格の管理と設定を行います。

### コマンド一覧

```bash
# 初期化（プロジェクトに .agents/ccpersona.json を作成）
ccpersona config init

# 利用可能な人格一覧
ccpersona persona list

# 現在の設定を表示
ccpersona config show

# 人格を設定
ccpersona config set-persona zundamon

# 人格の詳細を表示
ccpersona persona show zundamon

# 新しい人格を作成（存在しない場合は作成する）
ccpersona persona edit my_persona

# 人格を編集
ccpersona persona edit zundamon

# グローバル設定
ccpersona config edit --global
```

### 主要機能

1. **パーサ**: persona.json の読み書き
2. **バリデーション**: 設定値の妥当性チェック
3. **テンプレート**: 新規人格作成時のテンプレート提供
4. **フック連携**: UserPromptSubmit フックからの呼び出しに対応

## 実装優先順位

1. **Phase 1: 基本機能**
   - persona.json のパーサ実装
   - UserPromptSubmit フックの実装
   - 基本的な人格定義ファイル（default, zundamon）の作成

2. **Phase 2: CLI基本コマンド**
   - init, list, current, set コマンドの実装
   - グローバル gitignore への追加機能

3. **Phase 3: 高度な機能**
   - create, edit コマンド
   - 音声設定との連携
   - 自動ルーティング機能（プロジェクトタイプによる自動選択）

## セキュリティ考慮事項

- persona.json はプロジェクトごとに `.gitignore` に追加（グローバル設定）
- 人格定義ファイルには機密情報を含めない
- フックスクリプトの実行権限を適切に設定

## 互換性

- Claude Code の将来のアップデートに影響されないよう、独立したファイルで管理
- 標準の settings.json は変更せず、別ファイル（persona.json）を使用
- フォールバック機能により、設定がない場合もエラーにならない

## テスト計画

1. 各種人格定義ファイルでの動作確認
2. フック動作の検証
3. CLI コマンドの結合テスト
4. エラーケースの処理確認

## 今後の拡張案

- 時間帯による人格切り替え
- ユーザーの mood 検出による動的切り替え
- 複数人格の組み合わせ（ミックス機能）
- Web UI での人格管理
