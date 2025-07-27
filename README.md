# ccpersona - Claude Code Persona System

Claude Code のセッションごとに異なる「人格」を自動的に適用するシステムです。プロジェクトの特性に応じて、適切な人格（口調、考え方、専門性など）を自動的に選択・適用します。

## 特徴

- 🎭 **プロジェクトごとの人格設定** - 各プロジェクトに最適な人格を設定
- 🔄 **自動適用** - Claude Code セッション開始時に自動的に人格を適用
- 📝 **カスタマイズ可能** - 独自の人格を簡単に作成・編集
- 🎯 **一貫性のある対話** - プロジェクト全体で統一された応答スタイル

## インストール

### Homebrew (推奨)

```bash
brew tap daikw/tap
brew install ccpersona
```

### Go でビルド

```bash
git clone https://github.com/daikw/ccpersona.git
cd ccpersona
make build
make install
```

### リリースバイナリ

[Releases](https://github.com/daikw/ccpersona/releases) ページから最新のバイナリをダウンロードしてください。

## クイックスタート

1. **プロジェクトで初期化**

```bash
cd your-project
ccpersona init
```

2. **人格を設定**

```bash
# 利用可能な人格を確認
ccpersona list

# 人格を設定
ccpersona set zundamon
```

3. **Claude Code でフックを設定**

Claude Code の設定で UserPromptSubmit フックとして `ccpersona hook` を登録します。

```json
// Claude Code 設定例
{
  "hooks": {
    "user-prompt-submit": "ccpersona hook"
  }
}
```

これで、Claude Code セッション開始時に自動的に人格が適用されます。

## 使い方

### 基本コマンド

```bash
# プロジェクトで人格設定を初期化
ccpersona init

# 利用可能な人格一覧を表示
ccpersona list

# 現在の人格を表示
ccpersona current

# 人格を設定
ccpersona set <persona-name>

# 人格の詳細を表示
ccpersona show <persona-name>

# 新しい人格を作成
ccpersona create <persona-name>

# 人格を編集
ccpersona edit <persona-name>

# Claude Code でフックとして使用
ccpersona hook

# 音声読み上げ（最新のアシスタントメッセージを読み上げ）
ccpersona voice

# 音声読み上げオプション
ccpersona voice --mode full_text --engine voicevox
```

### 人格の作成

新しい人格を作成するには：

```bash
ccpersona create my-persona
ccpersona edit my-persona
```

エディタが開いて、人格の定義を編集できます。

## 人格定義ファイルの構造

人格は Markdown ファイルで定義され、以下の要素を含みます：

```markdown
# 人格: 名前

## 口調
話し方のスタイルを定義

## 考え方
問題解決のアプローチを定義

## 価値観
重視する価値観を定義

## 専門性・得意分野
特定の技術や分野への専門性

## 対話スタイル
質問への答え方、説明の仕方

## 感情表現パターン
喜怒哀楽の表現方法（オプション）
```

### サンプル人格

- **default** - 標準的で丁寧な技術者
- **zundamon** - 明るく元気なずんだもん
- **strict_engineer** - 厳格で効率重視のエンジニア

## プロジェクト設定

各プロジェクトの `.claude/persona.json` で設定を管理：

```json
{
  "name": "zundamon",
  "voice": {
    "engine": "voicevox",
    "speaker_id": 3
  },
  "override_global": true,
  "custom_instructions": "このプロジェクト固有の追加指示"
}
```

## 人格ファイルの保存場所

- グローバル人格: `~/.claude/personas/`
- プロジェクト設定: `<project>/.claude/persona.json`

## 開発

### 必要な環境

- Go 1.21 以上
- Make

### ビルド

```bash
make build
```

### テスト

```bash
make test
```

### リリース

```bash
make build-all
```

## 技術的な詳細

### フックの仕組み

ccpersona は Claude Code の UserPromptSubmit フックとして動作します：

1. Claude Code の設定で `ccpersona hook` をフックコマンドとして登録
2. Claude Code セッション開始時に `ccpersona hook` が実行される
3. ccpersona がプロジェクトの `.claude/persona.json` を読み込み
4. 設定された人格を自動的に適用

この設計により：
- シンプルな設定（brew install 後すぐ使える）
- クロスプラットフォーム対応（Windows/Mac/Linux）
- 堅牢なエラーハンドリング
- セッション追跡機能
- 高度なカスタマイズが可能

## ライセンス

MIT License

## 貢献

Issue や Pull Request を歓迎します！

## Acknowledgments

- [urfave/cli](https://github.com/urfave/cli) - CLI framework
- [zerolog](https://github.com/rs/zerolog) - Structured logging