# Hook Migration: シェルスクリプトから Go バイナリへ

## 概要

persona_router.sh のロジックを ccpersona バイナリに移行することで、以下のメリットが得られます。

## 移行前後の比較

### 移行前 (persona_router.sh)
- 61行のシェルスクリプト
- POSIX シェル依存
- エラーハンドリングが限定的
- Windows での動作に制限

### 移行後 (persona_router_simple.sh + ccpersona hook)
- 16行の簡潔なシェルスクリプト
- ロジックは Go で実装（session.go）
- 堅牢なエラーハンドリング
- クロスプラットフォーム対応

## アーキテクチャ

```
┌─────────────────────┐
│ user-prompt-submit  │
│      hook           │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│ persona_router_     │  <- 簡略化: ccpersona を呼び出すだけ
│    simple.sh        │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  ccpersona hook     │  <- ロジックはここに集約
│  (Go バイナリ)      │
└─────────────────────┘
           │
           ├── セッション管理
           ├── 設定読み込み
           └── Persona 適用
```

## 実装の詳細

### session.go の主な機能

1. **SessionManager** - セッション追跡
   - セッションIDの管理
   - マーカーファイルの作成/確認
   - 古いセッションのクリーンアップ

2. **HandleSessionStart** - メインエントリポイント
   - 新規セッションの判定
   - プロジェクト設定の読み込み
   - Persona の自動適用

### 移行によるコード改善

#### エラーハンドリング
```go
// Go: 詳細なエラー情報
if err := manager.ApplyPersona(config.Name); err != nil {
    return fmt.Errorf("failed to apply persona: %w", err)
}
```

#### クロスプラットフォーム対応
```go
// Go: OS に依存しないパス処理
filepath.Join(sm.homeDir, ".claude", fmt.Sprintf(".session_%s", sm.sessionID))
```

## 使用方法

### 1. ビルド
```bash
make build
```

### 2. フックのインストール
```bash
ccpersona install-hook
```

### 3. 動作確認
```bash
# セッション開始をシミュレート
CLAUDE_SESSION_ID=test-123 ccpersona hook
```

## テスト

セッション管理の包括的なテストを実装：

```bash
go test -v ./internal/persona -run TestSession
```

テストカバレッジ：
- セッションマネージャーの作成
- 新規/既存セッションの判定
- 古いセッションのクリーンアップ
- Persona 適用の統合テスト

## 今後の拡張性

Go 実装により、以下の機能追加が容易に：

1. **並行処理** - 複数セッションの同時処理
2. **設定のキャッシュ** - パフォーマンス向上
3. **詳細なロギング** - デバッグ機能の強化
4. **プラグインシステム** - カスタムフックの追加
5. **API サーバー** - リモート管理機能

## まとめ

シェルスクリプトから Go バイナリへの移行により：
- コードの保守性が向上
- エラーハンドリングが改善
- クロスプラットフォーム対応が実現
- テスタビリティが向上
- 将来の機能拡張が容易に