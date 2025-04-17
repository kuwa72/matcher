package matcher

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// 現実的なデータ構造
type TestData struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Active    bool      `json:"active"`
	Tags      []string  `json:"tags"`
	Address   struct {
		Street     string `json:"street"`
		City       string `json:"city"`
		State      string `json:"state"`
		PostalCode string `json:"postal_code"`
		Country    string `json:"country"`
	} `json:"address"`
	Phone      string  `json:"phone"`
	Website    string  `json:"website"`
	Company    string  `json:"company"`
	Department string  `json:"department"`
	JobTitle   string  `json:"job_title"`
	Salary     float64 `json:"salary"`
	Score      float64 `json:"score"`
	Status     string  `json:"status"`
	Notes      string  `json:"notes"`
	Path       string  `json:"path"`
}

// テストデータ生成用の定数
var (
	firstNames = []string{"John", "Jane", "Michael", "Emily", "David", "Sarah", "Robert", "Lisa", "William", "Mary"}
	lastNames  = []string{"Smith", "Johnson", "Brown", "Davis", "Wilson", "Miller", "Moore", "Taylor", "Anderson", "Thomas"}
	domains    = []string{"example.com", "test.org", "mail.net", "company.io", "service.co"}
	cities     = []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix", "Philadelphia", "San Antonio", "San Diego", "Dallas", "San Jose"}
	states     = []string{"NY", "CA", "IL", "TX", "AZ", "PA", "FL", "OH", "MI", "GA"}
	countries  = []string{"USA", "Canada", "UK", "Australia", "Germany", "France", "Japan", "China", "Brazil", "India"}
	companies  = []string{"Acme Corp", "Globex", "Initech", "Umbrella Corp", "Stark Industries", "Wayne Enterprises", "Cyberdyne Systems", "Soylent Corp", "Massive Dynamic", "Oscorp"}
	departments = []string{"Engineering", "Marketing", "Sales", "HR", "Finance", "Operations", "Research", "Development", "Support", "Legal"}
	jobTitles   = []string{"Engineer", "Manager", "Director", "Analyst", "Specialist", "Coordinator", "Administrator", "Supervisor", "Executive", "Consultant"}
	statuses    = []string{"Active", "Inactive", "Pending", "Suspended", "Reviewing", "Approved", "Rejected", "On Hold", "In Progress", "Completed"}
	tags        = []string{"important", "urgent", "review", "personal", "work", "family", "health", "finance", "education", "travel"}
	paths       = []string{"/api/v1/users", "/api/v1/products", "/api/v2/orders", "/dashboard", "/profile", "/settings", "/reports", "/analytics", "/help", "/support"}
)

// ランダムなテストデータを生成
func generateTestData(count int) []TestData {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	data := make([]TestData, count)
	
	for i := 0; i < count; i++ {
		firstName := firstNames[r.Intn(len(firstNames))]
		lastName := lastNames[r.Intn(len(lastNames))]
		domain := domains[r.Intn(len(domains))]
		
		data[i] = TestData{
			ID:        fmt.Sprintf("user-%d", i),
			Name:      firstName + " " + lastName,
			Email:     fmt.Sprintf("%s.%s@%s", firstName, lastName, domain),
			Age:       20 + r.Intn(50),
			CreatedAt: time.Now().Add(-time.Duration(r.Intn(1000)) * 24 * time.Hour),
			UpdatedAt: time.Now().Add(-time.Duration(r.Intn(100)) * 24 * time.Hour),
			Active:    r.Intn(2) == 1,
			Tags:      getRandomSubset(tags, 1+r.Intn(5), r),
			Phone:     fmt.Sprintf("+1-%d-%d-%d", 100+r.Intn(900), 100+r.Intn(900), 1000+r.Intn(9000)),
			Website:   fmt.Sprintf("https://www.%s.%s", lastName, domain),
			Company:   companies[r.Intn(len(companies))],
			Department: departments[r.Intn(len(departments))],
			JobTitle:   jobTitles[r.Intn(len(jobTitles))],
			Salary:     50000 + r.Float64()*100000,
			Score:      r.Float64() * 100,
			Status:     statuses[r.Intn(len(statuses))],
			Notes:      "Some notes about the user",
			Path:       paths[r.Intn(len(paths))],
		}
		
		data[i].Address.Street = fmt.Sprintf("%d %s St", 100+r.Intn(9000), lastName)
		data[i].Address.City = cities[r.Intn(len(cities))]
		data[i].Address.State = states[r.Intn(len(states))]
		data[i].Address.PostalCode = fmt.Sprintf("%d", 10000+r.Intn(90000))
		data[i].Address.Country = countries[r.Intn(len(countries))]
	}
	
	return data
}

// ランダムなサブセットを取得
func getRandomSubset(items []string, count int, r *rand.Rand) []string {
	if count >= len(items) {
		return items
	}
	
	result := make([]string, count)
	indices := r.Perm(len(items))
	
	for i := 0; i < count; i++ {
		result[i] = items[indices[i]]
	}
	
	return result
}

// 複雑なクエリを使用したベンチマーク
func BenchmarkComplexQueryWithLargeDataset(b *testing.B) {
	b.ReportAllocs()
	
	// 1万件のテストデータを生成
	dataCount := 10000
	testData := generateTestData(dataCount)
	
	// 複雑なクエリ（正規表現とグルーピングを含む）
	complexQuery := `(name = /^J.*/ OR department = "Engineering") AND (age > 30 AND salary >= 70000) AND (status = "Active" OR status = "Pending") AND path = /\/api\/v[0-9]\/.*/ AND score > 50`
	
	// マッチャーを作成
	matcher, err := NewMatcher(complexQuery)
	if err != nil {
		b.Fatalf("Failed to create matcher: %v", err)
	}
	
	// タイムアウト付きのコンテキスト
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// 各テストデータをJSON文字列に変換
	jsonData := make([]string, dataCount)
	for i, data := range testData {
		bytes, err := json.Marshal(data)
		if err != nil {
			b.Fatalf("Failed to marshal test data: %v", err)
		}
		jsonData[i] = string(bytes)
	}
	
	// 一致したアイテムの数をカウント（事前実行）
	matchCount := 0
	for _, jsonStr := range jsonData {
		var ctx Context
		err := json.Unmarshal([]byte(jsonStr), &ctx)
		if err != nil {
			b.Fatalf("Failed to unmarshal JSON: %v", err)
		}
		
		match, err := matcher.Test(&ctx)
		if err != nil {
			b.Fatalf("Test failed: %v", err)
		}
		
		if match {
			matchCount++
		}
	}
	
	b.Logf("Query matched %d out of %d items (%.2f%%)", matchCount, dataCount, float64(matchCount)/float64(dataCount)*100)
	
	// ベンチマーク実行
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < dataCount; j++ {
			var ctx Context
			err := json.Unmarshal([]byte(jsonData[j]), &ctx)
			if err != nil {
				b.Fatalf("Failed to unmarshal JSON: %v", err)
			}
			
			_, err = matcher.TestWithContext(ctxTimeout, &ctx)
			if err != nil {
				b.Fatalf("Test failed: %v", err)
			}
		}
	}
}

// より現実的なユースケースのベンチマーク（フィルタリング）
func BenchmarkFilteringWithLargeDataset(b *testing.B) {
	b.ReportAllocs()
	
	// 1万件のテストデータを生成
	dataCount := 10000
	testData := generateTestData(dataCount)
	
	// 複数のフィルタリングクエリ
	queries := []string{
		`age > 40 AND salary > 80000`,
		`name = /^[JM].*/ AND active = TRUE`,
		`(department = "Engineering" OR department = "Development") AND status = "Active"`,
		`path = /\/api\/v1\/.*/`,
		`company = "Acme Corp" AND score > 75`,
		`status = "Active" AND age < 35`,
	}
	
	// 各クエリに対してマッチャーを作成
	matchers := make([]*Matcher, len(queries))
	for i, query := range queries {
		matcher, err := NewMatcher(query)
		if err != nil {
			b.Fatalf("Failed to create matcher for query '%s': %v", query, err)
		}
		matchers[i] = matcher
	}
	
	// 各テストデータをJSON文字列に変換
	jsonData := make([]string, dataCount)
	for i, data := range testData {
		bytes, err := json.Marshal(data)
		if err != nil {
			b.Fatalf("Failed to marshal test data: %v", err)
		}
		jsonData[i] = string(bytes)
	}
	
	// ベンチマーク実行
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, matcher := range matchers {
			matchCount := 0
			for j := 0; j < dataCount; j++ {
				var ctx Context
				err := json.Unmarshal([]byte(jsonData[j]), &ctx)
				if err != nil {
					b.Fatalf("Failed to unmarshal JSON: %v", err)
				}
				
				match, err := matcher.Test(&ctx)
				if err != nil {
					b.Fatalf("Test failed: %v", err)
				}
				
				if match {
					matchCount++
				}
			}
		}
	}
}
