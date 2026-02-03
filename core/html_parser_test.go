package core

import (
	"bytes"
	"testing"
)

func TestInjectJavaScriptHTML(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		script   string
		location string
		wantErr  bool
		contains string
	}{
		{
			name:     "inject into head",
			html:     "<html><head><title>Test</title></head><body>Content</body></html>",
			script:   "console.log('test');",
			location: "head",
			wantErr:  false,
			contains: "<head><title>Test</title><script type=\"text/javascript\">console.log('test');</script></head>",
		},
		{
			name:     "inject into body_top",
			html:     "<html><head></head><body><div>Content</div></body></html>",
			script:   "alert('top');",
			location: "body_top",
			wantErr:  false,
			contains: "<body><script type=\"text/javascript\">alert('top');</script><div>Content</div></body>",
		},
		{
			name:     "inject into body_bottom",
			html:     "<html><head></head><body><div>Content</div></body></html>",
			script:   "alert('bottom');",
			location: "body_bottom",
			wantErr:  false,
			contains: "<div>Content</div><script type=\"text/javascript\">alert('bottom');</script></body>",
		},
		{
			name:     "invalid location",
			html:     "<html><head></head><body></body></html>",
			script:   "test();",
			location: "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := InjectJavaScriptHTML([]byte(tt.html), tt.script, tt.location)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("InjectJavaScriptHTML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.contains != "" {
				if !bytes.Contains(result, []byte(tt.contains)) {
					t.Errorf("InjectJavaScriptHTML() result does not contain expected content.\nGot: %s\nWant to contain: %s", string(result), tt.contains)
				}
			}
		})
	}
}

func TestHasHTMLStructure(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "full HTML document",
			content: "<html><head></head><body></body></html>",
			want:    true,
		},
		{
			name:    "HTML with only body",
			content: "<html><body>test</body></html>",
			want:    true,
		},
		{
			name:    "no HTML structure",
			content: "plain text content",
			want:    false,
		},
		{
			name:    "JSON content",
			content: "{\"key\": \"value\"}",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasHTMLStructure([]byte(tt.content)); got != tt.want {
				t.Errorf("HasHTMLStructure() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestObfuscateJavaScript(t *testing.T) {
	script := "var test = 'hello';\nconsole.log(test);"
	
	tests := []struct {
		name  string
		level string
	}{
		{"off", "off"},
		{"low", "low"},
		{"medium", "medium"},
		{"high", "high"},
		{"ultra", "ultra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ObfuscateJavaScript(script, tt.level)
			if result == "" {
				t.Error("ObfuscateJavaScript() returned empty string")
			}
			
			// For "off" level, should return original
			if tt.level == "off" && result != script {
				t.Errorf("ObfuscateJavaScript(off) should return original script")
			}
			
			// For "low" level, should have no newlines
			if tt.level == "low" && bytes.Contains([]byte(result), []byte("\n")) {
				t.Errorf("ObfuscateJavaScript(low) should remove newlines")
			}
		})
	}
}
