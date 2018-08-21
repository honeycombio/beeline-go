package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnynethttp"
)

func main() {
	// Initialize beeline. The only required field is WriteKey.
	beeline.Init(beeline.Config{
		WriteKey:    "abcabc123123abcabc",
		Dataset:     "http+sql",
		ServiceName: "sample app",
		// SamplerHook: sampler,
		// PresendHook: presend,
		// for demonstration, send the event to STDOUT instead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		STDOUT: true,
	})

	globalmux := http.NewServeMux()
	globalmux.HandleFunc("/hello/", hello)

	// wrap the globalmux with the honeycomb middleware to send one event per
	// request
	log.Fatal(http.ListenAndServe(":8080", hnynethttp.WrapHandler(globalmux)))
}

func hello(w http.ResponseWriter, r *http.Request) {
	beeline.AddField(r.Context(), "email", "one@two.com")
	bigJob(r.Context())
	outboundCall(r.Context())
	// send our response to the caller
	io.WriteString(w, fmt.Sprintf("Hello world!\n"))
}

// bigJob is going to take a long time and do lots of interesting work. It
// should get its own span.
func bigJob(ctx context.Context) {
	ctx, span := beeline.StartSpan(ctx, "bigJob")
	defer span.Finish()
	beeline.AddField(ctx, "m1", 5.67)
	beeline.AddField(ctx, "m2", 8.90)
	time.Sleep(600 * time.Millisecond)
	// this job also discovered something that's relevant to the whole trace
	beeline.AddFieldToTrace(ctx, "vip_user", true)
}

// outboundCall demonstrates wrapping an outbound HTTP client
func outboundCall(ctx context.Context) {
	// let's make an outbound HTTP call
	client := &http.Client{
		// Transport: hnynethttp.WrapRoundTripper(http.DefaultTransport),
		Timeout: time.Second * 5,
	}
	req, _ := http.NewRequest(http.MethodGet, "http://scooterlabs.com/echo.json", strings.NewReader(""))
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		bod, _ := ioutil.ReadAll(resp.Body)
		// data, _ := base64.StdEncoding.DecodeString(string(bod))
		beeline.AddField(ctx, "resp.body", string(bod))
	}
}

func presend(fields map[string]interface{}) {
	// If the email address field exists, add a field representing the
	// domain of the user's email address and hash the original email
	if email, ok := fields["app.email"]; ok {
		if emailStr, ok := email.(string); ok {
			splitEmail := strings.SplitN(emailStr, "@", 2)
			if len(splitEmail) == 2 {
				domain := splitEmail[1]
				fields["doamin"] = domain
			}
			// then hash the email so it is obscured
			hashedEmail := sha256.Sum256([]byte(fmt.Sprintf("%v", emailStr)))
			fields["app.email"] = fmt.Sprintf("%x", hashedEmail)
		}
	}
}

func sampler(fields map[string]interface{}) (bool, int) {
	// example sampler that samples at 1/3 when the "m1" field is present and
	// 1/2 when it is absent
	var sampleRate = 2
	if _, ok := fields["app.m1"]; ok {
		sampleRate = 3
	}
	if rand.Intn(sampleRate) == 0 {
		// keep the event!
		return true, sampleRate
	}
	// sample rate here doesn't matter because the event is going to get
	// dropped
	return false, 0
}
