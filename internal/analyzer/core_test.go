package analyzer

import (
	"reflect"
	"testing"
)

func Test_buildAllowedPunctuation(t *testing.T) {
	type args struct {
		cfg Config
	}
	tests := []struct {
		name string
		args args
		want map[rune]bool
	}{
		{
			name: "empty allowed punctuation",
			args: args{cfg: Config{AllowedPunctuation: ""}},
			want: map[rune]bool{' ': true},
		},
		{
			name: "some allowed punctuation",
			args: args{cfg: Config{AllowedPunctuation: ".,!?"}},
			want: map[rune]bool{'.': true, ',': true, '!': true, '?': true, ' ': true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildAllowedPunctuation(tt.args.cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildAllowedPunctuation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkDisallowedSymbols(t *testing.T) {
	type args struct {
		message string
		cfg     Config
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 string
	}{
		{
			name:  "no disallowed symbols",
			args:  args{message: "hello world", cfg: Config{AllowedPunctuation: ".,"}},
			want:  false,
			want1: "",
		},
		{
			name:  "disallowed symbol",
			args:  args{message: "hello@world", cfg: Config{AllowedPunctuation: ".,"}},
			want:  true,
			want1: "@",
		},
		{
			name:  "allowed punctuation",
			args:  args{message: "hello, world!", cfg: Config{AllowedPunctuation: ".,!"}},
			want:  false,
			want1: "",
		},
		{
			name:  "with newline",
			args:  args{message: "hello\nworld", cfg: Config{AllowedPunctuation: ".,"}},
			want:  true,
			want1: "\n",
		},
		{
			name:  "with symbol",
			args:  args{message: "hello©world", cfg: Config{AllowedPunctuation: ".,"}},
			want:  true,
			want1: "©",
		},
		{
			name:  "with disallowed punctuation",
			args:  args{message: "hello;world", cfg: Config{AllowedPunctuation: ".,"}},
			want:  true,
			want1: ";",
		},
		{
			name:  "empty message",
			args:  args{message: "", cfg: Config{AllowedPunctuation: ".,"}},
			want:  false,
			want1: "",
		},
		{
			name:  "only letters and digits",
			args:  args{message: "hello123world", cfg: Config{AllowedPunctuation: ".,"}},
			want:  false,
			want1: "",
		},
		{
			name:  "multiple disallowed, first one",
			args:  args{message: "hello@world!", cfg: Config{AllowedPunctuation: ".,"}},
			want:  true,
			want1: "@",
		},
		{
			name:  "with tab",
			args:  args{message: "hello\tworld", cfg: Config{AllowedPunctuation: ".,"}},
			want:  true,
			want1: "\t",
		},
		{
			name:  "with allowed and disallowed",
			args:  args{message: "hello, world; bad", cfg: Config{AllowedPunctuation: ".,"}},
			want:  true,
			want1: ";",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := checkDisallowedSymbols(tt.args.message, tt.args.cfg)
			if got != tt.want {
				t.Errorf("checkDisallowedSymbols() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("checkDisallowedSymbols() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_checkEnglishOnly(t *testing.T) {
	type args struct {
		message string
		cfg     Config
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 string
	}{
		{
			name:  "english only",
			args:  args{message: "hello world", cfg: Config{}},
			want:  false,
			want1: "",
		},
		{
			name:  "with non-latin letters",
			args:  args{message: "hello мир", cfg: Config{}},
			want:  true,
			want1: "hello ",
		},
		{
			name:  "empty message",
			args:  args{message: "", cfg: Config{}},
			want:  false,
			want1: "",
		},
		{
			name:  "numbers and punctuation",
			args:  args{message: "hello123!", cfg: Config{}},
			want:  false,
			want1: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := checkEnglishOnly(tt.args.message, tt.args.cfg)
			if got != tt.want {
				t.Errorf("checkEnglishOnly() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("checkEnglishOnly() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_checkSensitiveKeys(t *testing.T) {
	type args struct {
		message string
		cfg     Config
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 string
	}{
		{
			name:  "no sensitive keys",
			args:  args{message: "hello world", cfg: Config{}},
			want:  false,
			want1: "",
		},
		{
			name:  "contains password",
			args:  args{message: "user password is secret", cfg: Config{}},
			want:  true,
			want1: "password",
		},
		{
			name:  "contains api_key",
			args:  args{message: "api_key: 123", cfg: Config{}},
			want:  true,
			want1: "api_key",
		},
		{
			name:  "custom pattern match",
			args:  args{message: "custom sensitive", cfg: Config{CustomSensitivePatterns: []string{"custom"}}},
			want:  true,
			want1: "custom",
		},
		{
			name:  "no match with custom",
			args:  args{message: "normal message", cfg: Config{CustomSensitivePatterns: []string{"custom"}}},
			want:  false,
			want1: "",
		},
		{
			name:  "regex match secret",
			args:  args{message: "this is a secret message", cfg: Config{CustomSensitivePatterns: []string{"secret"}}},
			want:  true,
			want1: "secret",
		},
		{
			name:  "regex match with case insensitive",
			args:  args{message: "This is a SECRET", cfg: Config{CustomSensitivePatterns: []string{"(?i)secret"}}},
			want:  true,
			want1: "secret",
		},
		{
			name:  "regex match start with api",
			args:  args{message: "api_key here", cfg: Config{CustomSensitivePatterns: []string{"^api"}}},
			want:  true,
			want1: "api_key",
		},
		{
			name:  "regex no match",
			args:  args{message: "normal text", cfg: Config{CustomSensitivePatterns: []string{"^api"}}},
			want:  false,
			want1: "",
		},
		{
			name:  "regex with groups",
			args:  args{message: "token: abc123", cfg: Config{CustomSensitivePatterns: []string{"token:\\s*\\w+"}}},
			want:  true,
			want1: "token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := checkSensitiveKeys(tt.args.message, tt.args.cfg)
			if got != tt.want {
				t.Errorf("checkSensitiveKeys() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("checkSensitiveKeys() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_checkStartWithLowercase(t *testing.T) {
	type args struct {
		message string
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 string
	}{
		{
			name:  "starts with uppercase",
			args:  args{message: "Hello world"},
			want:  true,
			want1: "hello world",
		},
		{
			name:  "starts with lowercase",
			args:  args{message: "hello world"},
			want:  false,
			want1: "",
		},
		{
			name:  "empty message",
			args:  args{message: ""},
			want:  false,
			want1: "",
		},
		{
			name:  "starts with number",
			args:  args{message: "123 hello"},
			want:  false,
			want1: "",
		},
		{
			name:  "starts with space and uppercase",
			args:  args{message: " Hello"},
			want:  true,
			want1: "hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := checkStartWithLowercase(tt.args.message)
			if got != tt.want {
				t.Errorf("checkStartWithLowercase() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("checkStartWithLowercase() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
