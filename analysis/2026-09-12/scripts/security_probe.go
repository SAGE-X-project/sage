// Reproduces review findings using only locally generated keys and in-memory HTTP messages.
package main
import("crypto/ed25519";"crypto/rand";"encoding/json";"fmt";"net/http";"time";"github.com/sage-x-project/sage/pkg/agent/core/rfc9421";"github.com/sage-x-project/sage/pkg/agent/session")
func must(e error){if e!=nil{panic(e)}}
func main(){
 pub,priv,e:=ed25519.GenerateKey(rand.Reader);must(e)
 req,e:=http.NewRequest("GET","https://tool.example/resource",nil);must(e)
 p:=&rfc9421.SignatureInputParams{CoveredComponents:[]string{`"@method"`,`"@target-uri"`,`"@authority"`},KeyID:"did:example:alice#1",Algorithm:"ed25519",Nonce:"unique-request-1"}
 g:=session.NewMemoryReplayGuard(20*time.Millisecond);defer g.Close();v:=rfc9421.NewHTTPVerifierWithReplayGuard(g)
 must(v.SignRequest(req,"sig1",p,priv));o:=rfc9421.StrictHTTPVerificationOptions()
 first:=v.VerifyRequest(req,pub,o);duplicate:=v.VerifyRequest(req,pub,o);time.Sleep(30*time.Millisecond);expired:=v.VerifyRequest(req,pub,o)
 // A fresh verifier per invocation forgets all prior requests (basic-demo pattern).
 a:=rfc9421.NewHTTPVerifier();b:=rfc9421.NewHTTPVerifier();defer a.Close();defer b.Close()
 fresh1:=a.VerifyRequest(req,pub,o);fresh2:=b.VerifyRequest(req,pub,o)
 // A response without signature;req passes strict verification for two distinct request nonces.
 req2:=req.Clone(req.Context());p.Nonce="unique-request-2";must(v.SignRequest(req2,"sig1",p,priv))
 resp:=&http.Response{StatusCode:200,Header:http.Header{},ContentLength:0}
 rp:=&rfc9421.SignatureInputParams{CoveredComponents:[]string{`"@status"`,`"@method";req`,`"@target-uri";req`,`"@authority";req`},KeyID:"did:example:tool#1",Algorithm:"ed25519",Created:time.Now().Unix()}
 must(v.SignResponse(resp,req,"sig1",rp,priv));ro:=rfc9421.StrictHTTPResponseVerificationOptions()
 response1:=v.VerifyResponse(resp,req,pub,ro);response2:=v.VerifyResponse(resp,req2,pub,ro)
 out:=map[string]any{"strict_accepts_missing_created_expires":first==nil,"same_guard_rejects_immediate_replay":duplicate!=nil,"strict_accepts_replay_after_short_guard_ttl":expired==nil,"per_request_verifier_accepts_replay":fresh1==nil&&fresh2==nil,"strict_response_accepts_different_request_nonce":response1==nil&&response2==nil,"scope":"synthetic local reproduction; short TTL is custom configuration, missing timestamp is default strict behavior"};bb,_:=json.MarshalIndent(out,"","  ");fmt.Println(string(bb))
}
