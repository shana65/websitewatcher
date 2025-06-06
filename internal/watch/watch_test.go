package watch

import (
	"log/slog"
	gohttp "net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/softplowman/websitewatcher/internal/config"
	"github.com/softplowman/websitewatcher/internal/http"
	"github.com/stretchr/testify/require"
)

func TestCheck(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		UserAgent     string
		ServerContent string
		ServerStatus  int
		WantContent   string
		WantStatus    int
	}{
		"Default check": {
			UserAgent:     "xxx",
			ServerContent: "test",
			ServerStatus:  gohttp.StatusOK,
			WantContent:   "test",
			WantStatus:    gohttp.StatusOK,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(gohttp.HandlerFunc(func(w gohttp.ResponseWriter, r *gohttp.Request) {
				if tc.UserAgent != "" && r.Header.Get("User-Agent") != tc.UserAgent {
					t.Errorf("CheckWatch() want Useragent %s, got %s", tc.UserAgent, r.Header.Get("User-Agent"))
				}
				w.WriteHeader(tc.ServerStatus)
				if _, err := w.Write([]byte(tc.ServerContent)); err != nil {
					t.Fatalf("Write() err = %s, want nil", err)
				}
			}))
			defer server.Close()

			logger := slog.New(slog.DiscardHandler)
			client, err := http.NewHTTPClient(logger, tc.UserAgent, 1*time.Second, nil)
			if err != nil {
				t.Fatalf("NewHTTPClient() got err=%s, want nil", err)
			}
			w := New(
				config.WatchConfig{
					Name: "Test",
					URL:  server.URL,
				},
				logger,
				client,
			)

			ret, err := w.doHTTP(t.Context())
			if err != nil {
				t.Fatalf("CheckWatch() got err=%s, want nil", err)
			}
			if ret.StatusCode != tc.WantStatus {
				t.Errorf("CheckWatch() got status %d, want %d", ret.StatusCode, tc.WantStatus)
			}
			contentString := string(ret.Body)
			if contentString != tc.WantContent {
				t.Errorf("CheckWatch() got content %s, want %s", contentString, tc.WantContent)
			}
		})
	}
}

func TestExtractBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{
			input: `
			<!DOCTYPE html>
<html>
<head>
<title>
Title of the document
</title>
</head>
<body>body content<p>more content</p></body>
</html>
`,
			want: `<body>bodycontent<p>morecontent</p></body>`,
		},
	}

	for i, tc := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()

			gotBytes, err := extractBody([]byte(tc.input))
			require.NoError(t, err)
			// remove all whitespaces and newlines for comparison
			// as the renderer intruoduces newlines and spaces
			re := regexp.MustCompile(`\s+`)
			out := re.ReplaceAll(gotBytes, []byte(""))
			got := string(out)
			if got != tc.want {
				t.Errorf("extractBody() got:\n%s, want:\n%s", got, tc.want)
			}
		})
	}
}
