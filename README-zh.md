# Matcher

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

> **强大、灵活且安全的 Go 数据结构查询语言**

Matcher 是一个高性能 Go 库，让你可以使用简单而强大的查询语言来过滤数据结构。它支持正则表达式模式、逻辑运算符和括号分组等复杂表达式——非常适合过滤 JSON 数据、内存集合，或在 API 中实现查询功能。

*[English](README.md) | [日本語](README-ja.md) | 中文*

## ✨ 亮点

- **直观的查询语言** - 易于学习和使用的类 SQL 语法
- **强大的正则表达式支持** - 具备完整正则表达式能力的字符串模式匹配
- **括号分组** - 精确控制，构建复杂的嵌套表达式
- **高性能** - 为速度优化，内存分配极少
- **内置安全性** - 防护 ReDoS 攻击和资源耗尽
- **Context 支持** - 通过超时取消长时间运行的操作

## 🚀 快速开始

### 安装

```bash
go get github.com/kuwa72/matcher
```

### 基本示例

```go
package main

import (
	"fmt"
	"github.com/kuwa72/matcher"
)

func main() {
	// 使用查询字符串创建匹配器
	m, err := matcher.NewMatcher(`name = "John" AND age > 30`)
	if err != nil {
		panic(err)
	}

	// 要测试的数据
	data := matcher.Context{
		"name": "John",
		"age":  35,
	}

	// 测试数据是否匹配查询
	result, err := m.Test(&data)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Match: %v\n", result) // 输出: Match: true
}
```

### CLI 工具

```bash
# 安装
go install github.com/kuwa72/matcher/matcher-cli@latest

# 基本用法
echo '{"name":"John","age":35}' | matcher-cli 'name = "John" AND age > 30'

# 调试输出
echo '{"name":"John","age":35}' | matcher-cli --debug 'name = "John" AND age > 30'
```

## 🔍 查询语言

Matcher 使用直观的查询语言，既易于学习，又足以满足复杂的过滤需求。

### 主要特性

* **逻辑运算符**: `AND`、`OR`、`NOT`（不区分大小写）
* **比较运算符**: `=`、`!=`、`<>`、`>`、`>=`、`<`、`<=`
* **成员运算符**: `IN`（值列表）、`CONTAINS`（子字符串 / 数组元素）
* **分组**: 括号 `()` 用于精确控制求值顺序
* **值类型**:
  * **数字**: 整数和浮点值
  * **字符串**: 用单引号或双引号括起来
  * **正则表达式**: 用 `/pattern/` 括起来的模式
  * **布尔值**: `TRUE` 或 `FALSE`（不区分大小写）
  * **NULL**: 用于 null 检查的特殊值

### 运算符优先级

1. 比较运算（`=`、`!=`、`IN`、`CONTAINS` 等）和 `NOT` 最先求值
2. `AND` 条件其次求值
3. `OR` 条件最后求值

### 📝 查询示例

```
# 简单的相等比较
age = 30

# 使用 AND 的多个条件
name = "John" AND age > 30 AND status = "active"

# 使用 OR 表示备选
country = "USA" OR country = "Canada"

# 用于分组的括号
(status = "pending" OR status = "approved") AND created_at > "2025-01-01"

# 使用 NOT 取反
NOT status = "archived"
NOT (country = "USA" OR country = "Canada")

# 使用 IN 和 CONTAINS 的成员判断
status IN ("active", "pending")
tags CONTAINS "golang"        # 字符串做子串匹配，数组做元素匹配

# 复杂的嵌套表达式
(category = "electronics" AND (price < 1000 OR rating > 4.5)) OR featured = TRUE

# 正则表达式匹配
email = /.*@gmail\.com$/    # 匹配 Gmail 地址
name = /^(John|Jane).*/     # 以 John 或 Jane 开头的名字

# 包含斜杠的正则表达式
path = /\/api\/v1\/.*/      # 匹配 API v1 路径
url = /https:\/\/.*/       # 匹配 HTTPS URL

# 组合所有条件
(name = /J.*/ OR department = "Engineering") AND 
(age > 30 AND salary >= 70000) AND 
(status = "Active" OR status = "Pending")
```

更多示例请参阅[测试文件](https://github.com/kuwa72/matcher/blob/main/parser_test.go)。

## 🔒 正则表达式支持

Matcher 为字符串值提供强大的正则表达式模式匹配，并内置安全保护。

### 🛡️ 安全特性

所有正则表达式操作都包含针对 ReDoS 攻击和资源耗尽的防护：

- **模式长度限制**: 每个模式最多 1000 个字符
- **复杂度限制**: 重复运算符（`*`、`+`、`{...}`、`?`、`|`）不超过 20 个
- **编译超时**: 100ms 超时防止灾难性回溯
- **异步处理**: 在独立 goroutine 中进行非阻塞编译

### 📋 语法

```
field = /pattern/   # 字段匹配模式时命中
field != /pattern/  # 字段不匹配模式时命中
field = /pattern/i  # 不区分大小写匹配（i 标志）
```

### 📌 重要说明

- 使用 Go 标准 `regexp` 包的语法
- 支持相等（`=`）和不等（`!=`、`<>`）运算符
- 仅适用于字符串值
- 用反斜杠转义斜杠（`\/`）
- 在闭合斜杠后加 `i` 表示不区分大小写（`/john/i`）

### 🌟 正则表达式示例

```go
// 邮箱验证
email = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/

// URL 路径匹配
path = /\/api\/v[0-9]\/users/

// 文件扩展名
filename = /\.(jpg|png|gif)$/

// 电话号码
phone = /^\+?[0-9]{10,15}$/

// 包含转义的复杂模式
url = /https:\/\/[^\/]+\/[^\/]+/
```

## ⚡ 性能

Matcher 专为高性能设计，即使面对大型数据集和复杂查询也能表现出色。

### 📊 基准测试结果

在 AMD Ryzen 9 5900HS 上测试 10,000 条记录（每条 20 个字段）：

#### 复杂查询性能

```
BenchmarkComplexQueryWithLargeDataset-16    5    215ms/op    47MB/op    1,095,634 allocs/op
```

**测试的查询：**
```
(name = /^J.*/ OR department = "Engineering") AND 
(age > 30 AND salary >= 70000) AND 
(status = "Active" OR status = "Pending") AND 
path = /\/api\/v[0-9]\/.*/ AND score > 50
```

#### 多条件过滤性能

```
BenchmarkFilteringWithLargeDataset-16    1    1,325ms/op    282MB/op    6,580,071 allocs/op
```

### 🔧 优化建议

1. **复用匹配器** - 创建一次，多次复用
2. **优先使用简单比较** - 尽可能使用 `=`、`>`、`<` 而非正则表达式
3. **优化查询顺序** - 在 AND 表达式中把更可能失败的条件放在前面
4. **限制正则表达式复杂度** - 更简单的模式性能更好

## 🔧 高级用法

### Context 支持

Matcher 支持 Go 的 context 包，用于超时和取消：

```go
// 创建带超时的 context
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// 使用 context 进行测试
result, err := matcher.TestWithContext(ctx, &data)
```

### JSON 集成

Matcher 可以与 JSON 数据无缝协作：

```go
// 解析 JSON 数据
var data matcher.Context
json.Unmarshal([]byte(`{"name":"John","age":35}`), &data)

// 创建匹配器
matcher, _ := matcher.NewMatcher(`name = "John" AND age > 30`)

// 对 JSON 数据进行测试
result, _ := matcher.Test(&data)
```

## 📦 环境要求

- Go 1.22 或更高版本

## 📄 许可证

本项目采用 MIT 许可证 - 详情请参阅 [LICENSE](LICENSE) 文件。

## 👥 贡献

欢迎贡献！请随时提交 issue 或 pull request。
