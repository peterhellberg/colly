package colly

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"

	"github.com/gocolly/colly/debug"
)

var testServerPort = 31337
var testServerAddr = fmt.Sprintf("127.0.0.1:%d", testServerPort)
var testServerRootURL = fmt.Sprintf("http://%s/", testServerAddr)
var serverIndexResponse = []byte("hello world\n")
var robotsFile = `
User-agent: *
Allow: /allowed
Disallow: /disallowed
`

func init() {
	srv := &http.Server{}
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: testServerPort})
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(serverIndexResponse)
	})

	http.HandleFunc("/html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Conent-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<title>Test Page</title>
</head>
<body>
<h1>Hello World</h1>
<p class="description">This is a test page</p>
<p class="description">This is a test paragraph</p>
</body>
</html>
		`))
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Conent-Type", "text/html")
			w.Write([]byte(r.FormValue("name")))
		}
	})

	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(robotsFile))
	})

	http.HandleFunc("/allowed", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("allowed"))
	})

	http.HandleFunc("/disallowed", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("disallowed"))
	})

	http.Handle("/redirect", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/redirected/", http.StatusSeeOther)

	}))
	http.Handle("/redirected/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<a href="test">test</a>`)
	}))

	http.HandleFunc("/set_cookie", func(w http.ResponseWriter, r *http.Request) {
		c := &http.Cookie{Name: "test", Value: "testv", HttpOnly: false}
		http.SetCookie(w, c)
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	http.HandleFunc("/check_cookie", func(w http.ResponseWriter, r *http.Request) {
		cs := r.Cookies()
		if len(cs) != 1 || r.Cookies()[0].Value != "testv" {
			w.WriteHeader(500)
			w.Write([]byte("nok"))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	go func() {
		if err := srv.Serve(listener); err != nil {
			log.Printf("Httpserver: ListenAndServe() error: %s", err)
		}
	}()
}

func TestNewCollector(t *testing.T) {
	t.Run("Functional Options", func(t *testing.T) {
		t.Run("UserAgent", func(t *testing.T) {
			for _, ua := range []string{
				"foo",
				"bar",
			} {
				c := NewCollector(UserAgent(ua))

				if got, want := c.UserAgent, ua; got != want {
					t.Fatalf("c.UserAgent = %q, want %q", got, want)
				}
			}
		})

		t.Run("MaxDepth", func(t *testing.T) {
			for _, depth := range []int{
				12,
				34,
				0,
			} {
				c := NewCollector(MaxDepth(depth))

				if got, want := c.MaxDepth, depth; got != want {
					t.Fatalf("c.MaxDepth = %d, want %d", got, want)
				}
			}
		})

		t.Run("AllowedDomains", func(t *testing.T) {
			for _, domains := range [][]string{
				[]string{"example.com", "example.net"},
				[]string{"example.net"},
				[]string{},
				nil,
			} {
				c := NewCollector(AllowedDomains(domains...))

				if got, want := c.AllowedDomains, domains; !reflect.DeepEqual(got, want) {
					t.Fatalf("c.AllowedDomains = %q, want %q", got, want)
				}
			}
		})

		t.Run("DisallowedDomains", func(t *testing.T) {
			for _, domains := range [][]string{
				[]string{"example.com", "example.net"},
				[]string{"example.net"},
				[]string{},
				nil,
			} {
				c := NewCollector(DisallowedDomains(domains...))

				if got, want := c.DisallowedDomains, domains; !reflect.DeepEqual(got, want) {
					t.Fatalf("c.DisallowedDomains = %q, want %q", got, want)
				}
			}
		})

		t.Run("URLFilters", func(t *testing.T) {
			for _, filters := range [][]*regexp.Regexp{
				[]*regexp.Regexp{regexp.MustCompile(`\w+`)},
				[]*regexp.Regexp{regexp.MustCompile(`\d+`)},
				[]*regexp.Regexp{},
				nil,
			} {
				c := NewCollector(URLFilters(filters...))

				if got, want := c.URLFilters, filters; !reflect.DeepEqual(got, want) {
					t.Fatalf("c.URLFilters = %v, want %v", got, want)
				}
			}
		})

		t.Run("AllowURLRevisit", func(t *testing.T) {
			c := NewCollector(AllowURLRevisit())

			if !c.AllowURLRevisit {
				t.Fatal("c.AllowURLRevisit = false, want true")
			}
		})

		t.Run("MaxBodySize", func(t *testing.T) {
			for _, sizeInBytes := range []int{
				1024 * 1024,
				1024,
				0,
			} {
				c := NewCollector(MaxBodySize(sizeInBytes))

				if got, want := c.MaxBodySize, sizeInBytes; got != want {
					t.Fatalf("c.MaxBodySize = %d, want %d", got, want)
				}
			}
		})

		t.Run("CacheDir", func(t *testing.T) {
			for _, path := range []string{
				"/tmp/",
				"/var/cache/",
			} {
				c := NewCollector(CacheDir(path))

				if got, want := c.CacheDir, path; got != want {
					t.Fatalf("c.CacheDir = %q, want %q", got, want)
				}
			}
		})

		t.Run("IgnoreRobotsTxt", func(t *testing.T) {
			c := NewCollector(IgnoreRobotsTxt())

			if !c.IgnoreRobotsTxt {
				t.Fatal("c.IgnoreRobotsTxt = false, want true")
			}
		})

		t.Run("ID", func(t *testing.T) {
			for _, id := range []uint32{
				0,
				1,
				2,
			} {
				c := NewCollector(ID(id))

				if got, want := c.ID, id; got != want {
					t.Fatalf("c.ID = %d, want %d", got, want)
				}
			}
		})

		t.Run("DetectCharset", func(t *testing.T) {
			c := NewCollector(DetectCharset())

			if !c.DetectCharset {
				t.Fatal("c.DetectCharset = false, want true")
			}
		})

		t.Run("Debugger", func(t *testing.T) {
			d := &debug.LogDebugger{}
			c := NewCollector(Debugger(d))

			if got, want := c.debugger, d; got != want {
				t.Fatalf("c.debugger = %v, want %v", got, want)
			}
		})
	})
}

func TestCollectorVisit(t *testing.T) {
	c := NewCollector()

	onRequestCalled := false
	onResponseCalled := false
	onScrapedCalled := false

	c.OnRequest(func(r *Request) {
		onRequestCalled = true
		r.Ctx.Put("x", "y")
	})

	c.OnResponse(func(r *Response) {
		onResponseCalled = true

		if r.Ctx.Get("x") != "y" {
			t.Error("Failed to retrieve context value for key 'x'")
		}

		if !bytes.Equal(r.Body, serverIndexResponse) {
			t.Error("Response body does not match with the original content")
		}
	})

	c.OnScraped(func(r *Response) {
		if !onResponseCalled {
			t.Error("OnScraped called before OnResponse")
		}

		if !onRequestCalled {
			t.Error("OnScraped called before OnRequest")
		}

		onScrapedCalled = true
	})

	c.Visit(testServerRootURL)

	if !onRequestCalled {
		t.Error("Failed to call OnRequest callback")
	}

	if !onResponseCalled {
		t.Error("Failed to call OnResponse callback")
	}

	if !onScrapedCalled {
		t.Error("Failed to call OnScraped callback")
	}
}

func TestCollectorOnHTML(t *testing.T) {
	c := NewCollector()

	titleCallbackCalled := false
	paragraphCallbackCount := 0

	c.OnHTML("title", func(e *HTMLElement) {
		titleCallbackCalled = true
		if e.Text != "Test Page" {
			t.Error("Title element text does not match, got", e.Text)
		}
	})

	c.OnHTML("p", func(e *HTMLElement) {
		paragraphCallbackCount++
		if e.Attr("class") != "description" {
			t.Error("Failed to get paragraph's class attribute")
		}
	})

	c.OnHTML("body", func(e *HTMLElement) {
		if e.ChildAttr("p", "class") != "description" {
			t.Error("Invalid class value")
		}
		classes := e.ChildAttrs("p", "class")
		if len(classes) != 2 {
			t.Error("Invalid class values")
		}
	})

	c.Visit(testServerRootURL + "html")

	if !titleCallbackCalled {
		t.Error("Failed to call OnHTML callback for <title> tag")
	}

	if paragraphCallbackCount != 2 {
		t.Error("Failed to find all <p> tags")
	}
}

func TestCollectorURLRevisit(t *testing.T) {
	c := NewCollector()

	visitCount := 0

	c.OnRequest(func(r *Request) {
		visitCount++
	})

	c.Visit(testServerRootURL)
	c.Visit(testServerRootURL)

	if visitCount != 1 {
		t.Error("URL revisited")
	}

	c.AllowURLRevisit = true

	c.Visit(testServerRootURL)
	c.Visit(testServerRootURL)

	if visitCount != 3 {
		t.Error("URL not revisited")
	}
}

func TestCollectorPost(t *testing.T) {
	postValue := "hello"
	c := NewCollector()

	c.OnResponse(func(r *Response) {
		if postValue != string(r.Body) {
			t.Error("Failed to send data with POST")
		}
	})

	c.Post(testServerRootURL+"login", map[string]string{
		"name": postValue,
	})
}

func TestRedirect(t *testing.T) {
	c := NewCollector()
	c.OnHTML("a[href]", func(e *HTMLElement) {
		u := e.Request.AbsoluteURL(e.Attr("href"))
		if !strings.HasSuffix(u, "/redirected/test") {
			t.Error("Invalid URL after redirect: " + u)
		}
	})
	c.Visit(testServerRootURL + "redirect")
}

func TestCollectorCookies(t *testing.T) {
	c := NewCollector()

	if err := c.Visit(testServerRootURL + "set_cookie"); err != nil {
		t.Fatal(err)
	}

	if err := c.Visit(testServerRootURL + "check_cookie"); err != nil {
		t.Fatalf("Failed to use previously set cookies: %s", err)
	}
}

func BenchmarkVisit(b *testing.B) {
	c := NewCollector()
	c.OnHTML("p", func(_ *HTMLElement) {})

	for n := 0; n < b.N; n++ {
		c.Visit(fmt.Sprintf("%shtml?q=%d", testServerRootURL, n))
	}
}

func TestRobotsWhenAllowed(t *testing.T) {
	c := NewCollector()
	c.IgnoreRobotsTxt = false

	c.OnResponse(func(resp *Response) {
		if resp.StatusCode != 200 {
			t.Fatalf("Wrong response code: %d", resp.StatusCode)
		}
	})

	err := c.Visit(testServerRootURL + "allowed")

	if err != nil {
		t.Fatal(err)
	}
}

func TestRobotsWhenDisallowed(t *testing.T) {
	c := NewCollector()
	c.IgnoreRobotsTxt = false

	c.OnResponse(func(resp *Response) {
		t.Fatalf("Received response: %d", resp.StatusCode)
	})

	err := c.Visit(testServerRootURL + "disallowed")
	if err.Error() != "URL blocked by robots.txt" {
		t.Fatalf("wrong error message: %v", err)
	}
}

func TestIgnoreRobotsWhenDisallowed(t *testing.T) {
	c := NewCollector()
	c.IgnoreRobotsTxt = true

	c.OnResponse(func(resp *Response) {
		if resp.StatusCode != 200 {
			t.Fatalf("Wrong response code: %d", resp.StatusCode)
		}
	})

	err := c.Visit(testServerRootURL + "disallowed")

	if err != nil {
		t.Fatal(err)
	}

}

func TestHTMLElement(t *testing.T) {
	ctx := &Context{}
	resp := &Response{
		Request: &Request{
			Ctx: ctx,
		},
		Ctx: ctx,
	}

	in := `<a href="http://go-colly.org">Colly</a>`
	sel := "a[href]"
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer([]byte(in)))
	if err != nil {
		t.Fatal(err)
	}
	elements := []*HTMLElement{}
	doc.Find(sel).Each(func(i int, s *goquery.Selection) {
		for _, n := range s.Nodes {
			elements = append(elements, NewHTMLElementFromSelectionNode(resp, s, n))
		}
	})
	elementsLen := len(elements)
	if elementsLen != 1 {
		t.Errorf("element length mismatch. got %d, expected %d.\n", elementsLen, 1)
	}
	v := elements[0]
	if v.Name != "a" {
		t.Errorf("element tag mismatch. got %s, expected %s.\n", v.Name, "a")
	}
	if v.Text != "Colly" {
		t.Errorf("element content mismatch. got %s, expected %s.\n", v.Text, "Colly")
	}
	if v.Attr("href") != "http://go-colly.org" {
		t.Errorf("element href mismatch. got %s, expected %s.\n", v.Attr("href"), "http://go-colly.org")
	}
}
