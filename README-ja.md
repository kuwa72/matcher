# Matcher

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

> **強力でフレキシブル、かつ安全なGoデータ構造のためのクエリ言語**

Matcherは、シンプルかつ強力なクエリ言語を使用してGoのデータ構造をフィルタリングできる高性能ライブラリです。正規表現パターン、論理演算子、括弧によるグループ化をサポートし、JSONデータのフィルタリングやAPIでのクエリ機能の実装に最適です。

## ✨ 特徴

- **直感的なクエリ言語** - 学習と使用が容易なSQL風の構文
- **強力な正規表現サポート** - 完全な正規表現機能による文字列パターンマッチング
- **括弧によるグループ化** - 正確な制御で複雑なネストされた式を構築
- **高性能** - 最小限のメモリ割り当てで速度を最適化
- **セキュリティ内蔵** - ReDoS攻撃やリソース枯渇からの保護
- **コンテキストサポート** - タイムアウトによる長時間実行操作のキャンセル

## 🚀 クイックスタート

### インストール

```bash
go get github.com/kuwa72/matcher
```

### 基本的な例

```go
package main

import (
	"fmt"
	"github.com/kuwa72/matcher"
)

func main() {
	// クエリ文字列でマッチャーを作成
	m, err := matcher.NewMatcher(`name = "John" AND age > 30`)
	if err != nil {
		panic(err)
	}

	// テスト対象のデータ
	data := matcher.Context{
		"name": "John",
		"age":  35,
	}

	// データがクエリにマッチするかテスト
	result, err := m.Test(&data)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Match: %v\n", result) // 出力: Match: true
}
```

### CLIツール

```bash
# インストール
go install github.com/kuwa72/matcher/matcher-cli@latest

# 基本的な使用法
echo '{"name":"John","age":35}' | matcher-cli 'name = "John" AND age > 30'

# デバッグ出力
echo '{"name":"John","age":35}' | matcher-cli --debug 'name = "John" AND age > 30'
```

## 🔍 クエリ言語

Matcherは、複雑なフィルタリングニーズに対応できる直感的なクエリ言語を使用します。

### 主な機能

* **論理演算子**: `AND`, `OR` (大文字小文字を区別しない)
* **比較演算子**: `=`, `!=`, `<>`, `>`, `>=`, `<`, `<=`
* **グループ化**: 括弧 `()` で評価順序を正確に制御
* **値の型**:
  * **数値**: 整数と浮動小数点値
  * **文字列**: 単一引用符または二重引用符で囲む
  * **正規表現**: `/パターン/` で囲まれたパターン
  * **ブール値**: `TRUE` または `FALSE` (大文字小文字を区別しない)
  * **NULL**: null チェック用の特殊値

### 演算子の優先順位

1. 比較 (`=`, `!=` など) が最初に評価される
2. `AND` 条件が次に評価される
3. `OR` 条件が最後に評価される

### 📝 クエリ例

```
# 単純な等価比較
age = 30

# ANDによる複数条件
name = "John" AND age > 30 AND status = "active"

# OR による代替
country = "USA" OR country = "Canada"

# グループ化のための括弧
(status = "pending" OR status = "approved") AND created_at > "2025-01-01"

# 複雑なネストされた式
(category = "electronics" AND (price < 1000 OR rating > 4.5)) OR featured = TRUE

# 正規表現マッチング
email = /.*@gmail\.com$/    # Gmailアドレスにマッチ
name = /^(John|Jane).*/     # JohnまたはJaneで始まる名前

# スラッシュを含む正規表現
path = /\/api\/v1\/.*/      # API v1パスにマッチ
url = /https:\/\/.*/       # HTTPSのURLにマッチ

# すべてを組み合わせる
(name = /J.*/ OR department = "Engineering") AND 
(age > 30 AND salary >= 70000) AND 
(status = "Active" OR status = "Pending")
```

詳細な例については、[テストファイル](https://github.com/kuwa72/matcher/blob/main/parser_test.go)を参照してください。

## 🔒 正規表現サポート

Matcherは文字列値に対して強力な正規表現パターンマッチングを提供し、組み込みのセキュリティ保護機能を備えています。

### 🛡️ セキュリティ機能

すべての正規表現操作には、ReDoS攻撃やリソース枯渇からの保護が含まれています：

- **パターン長の制限**: パターンあたり最大1000文字
- **複雑さの制限**: 繰り返し演算子（`*`, `+`, `{...}`, `?`, `|`）は20個まで
- **コンパイルタイムアウト**: 100msのタイムアウトでカタストロフィックバックトラッキングを防止
- **非同期処理**: 別のゴルーチンでのノンブロッキングコンパイル

### 📋 構文

```
field = /pattern/   # フィールドがパターンにマッチする場合
field != /pattern/  # フィールドがパターンにマッチしない場合
```

### 📌 重要な注意点

- Goの標準 `regexp` パッケージの構文を使用
- 等価（`=`）および不等価（`!=`, `<>`）演算子で動作
- 文字列値にのみ適用
- スラッシュはバックスラッシュでエスケープ（`\/`）

### 🌟 正規表現の例

```go
// メール検証
email = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/

// URLパスマッチング
path = /\/api\/v[0-9]\/users/

// ファイル拡張子
filename = /\.(jpg|png|gif)$/

// 電話番号
phone = /^\+?[0-9]{10,15}$/

// エスケープを含む複雑なパターン
url = /https:\/\/[^\/]+\/[^\/]+/
```

## ⚡ パフォーマンス

Matcherは大規模なデータセットと複雑なクエリでも高いパフォーマンスを発揮するように設計されています。

### 📊 ベンチマーク結果

AMD Ryzen 9 5900HSで10,000レコード（各20フィールド）をテスト：

#### 複雑なクエリのパフォーマンス

```
BenchmarkComplexQueryWithLargeDataset-16    5    215ms/op    47MB/op    1,095,634 allocs/op
```

**テストしたクエリ：**
```
(name = /^J.*/ OR department = "Engineering") AND 
(age > 30 AND salary >= 70000) AND 
(status = "Active" OR status = "Pending") AND 
path = /\/api\/v[0-9]\/.*/ AND score > 50
```

#### 複数フィルターのパフォーマンス

```
BenchmarkFilteringWithLargeDataset-16    1    1,325ms/op    282MB/op    6,580,071 allocs/op
```

### 🔧 最適化のヒント

1. **マッチャーを再利用する** - 一度作成して何度も再利用
2. **単純な比較を優先する** - 可能な場合は正規表現の代わりに `=`, `>`, `<` を使用
3. **クエリの順序を最適化する** - AND式では失敗しやすい条件を最初に配置
4. **正規表現の複雑さを制限する** - シンプルなパターンの方がパフォーマンスが良い

## 🔧 高度な使用法

### コンテキストサポート

MatcherはタイムアウトとキャンセルのためにGoのcontextパッケージをサポートしています：

```go
// タイムアウト付きコンテキストを作成
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// コンテキスト付きでテスト
result, err := matcher.TestWithContext(ctx, &data)
```

### JSON統合

MatcherはJSONデータとシームレスに連携します：

```go
// JSONデータの解析
var data matcher.Context
json.Unmarshal([]byte(`{"name":"John","age":35}`), &data)

// マッチャーの作成
matcher, _ := matcher.NewMatcher(`name = "John" AND age > 30`)

// JSONデータに対するテスト
result, _ := matcher.Test(&data)
```

## 📦 要件

- Go 1.22 以上

## 📄 ライセンス

このプロジェクトはMITライセンスの下で提供されています - 詳細は[LICENSE](LICENSE)ファイルを参照してください。

## 👥 貢献

貢献は歓迎します！気軽にイシューを開いたり、プルリクエストを送信してください。
