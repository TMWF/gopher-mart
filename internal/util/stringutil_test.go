package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasSubstring(t *testing.T) {
	tests := []struct {
		name   string
		slice  []string
		substr string
		want   bool
	}{
		{
			name:   "Substring exists in the middle of a string",
			slice:  []string{"apple", "banana", "cherry"},
			substr: "nan",
			want:   true,
		},
		{
			name:   "Exact match",
			slice:  []string{"go", "rust", "cpp"},
			substr: "rust",
			want:   true,
		},
		{
			name:   "Substring at the beginning",
			slice:  []string{"frontend", "backend"},
			substr: "front",
			want:   true,
		},
		{
			name:   "Substring does not exist",
			slice:  []string{"hello", "world"},
			substr: "golang",
			want:   false,
		},
		{
			name:   "Case sensitivity check",
			slice:  []string{"Gopher"},
			substr: "gopher",
			want:   false, // strings.Contains чувствителен к регистру
		},
		{
			name:   "Empty slice",
			slice:  []string{},
			substr: "any",
			want:   false,
		},
		{
			name:   "Nil slice",
			slice:  nil,
			substr: "any",
			want:   false,
		},
		{
			name:   "Empty substring",
			slice:  []string{"apple", ""},
			substr: "",
			want:   true, // Поведение strings.Contains: любая строка содержит пустую подстроку
		},
		{
			name:   "Empty slice and empty substring",
			slice:  []string{},
			substr: "",
			want:   false, // Цикл не выполнится ни разу
		},
		{
			name:   "Slice with empty string",
			slice:  []string{""},
			substr: "a",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasSubstring(tt.slice, tt.substr)
			assert.Equal(t, tt.want, got, "HasSubstring(%v, %q)", tt.slice, tt.substr)
		})
	}
}

// Benchmark для проверки производительности (Senior practice)
func BenchmarkHasSubstring(b *testing.B) {
	slice := []string{"apple", "banana", "cherry", "date", "elderberry"}
	substr := "berry"

	for i := 0; i < b.N; i++ {
		HasSubstring(slice, substr)
	}
}
