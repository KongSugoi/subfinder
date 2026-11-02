package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"
)

// ====================== CONFIG ======================

type Config struct {
	Domains       []string `yaml:"domains"`
	Telegram      Telegram `yaml:"telegram"`
	ScheduleHours int      `yaml:"schedule_hours"`
	OutputDir     string   `yaml:"output_dir"`
	DataDir       string   `yaml:"data_dir"`
	Wordlist      string   `yaml:"wordlist"`
}

type Telegram struct {
	BotToken string `yaml:"bot_token"`
	ChatID   string `yaml:"chat_id"`
}

type SubdomainRecord struct {
	Subdomain string    `json:"subdomain"`
	IP        string    `json:"ip"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

type DomainResult struct {
	Domain    string            `json:"domain"`
	Records   []SubdomainRecord `json:"records"`
	ScannedAt time.Time         `json:"scanned_at"`
}

// ====================== GLOBAL ======================

var (
	cfg       Config
	toolsDir  = "tools"
	dataDir   string
	outputDir string
)

// Wordlist
var defaultWordlist = []string{
	"www", "mail", "ftp", "localhost", "webmail", "smtp", "webdisk", "pop", "ns1", "ns2",
	"cpanel", "whm", "autodiscover", "autoconfig", "m", "imap", "test", "ns", "blog",
	"pop3", "dev", "www2", "admin", "forum", "news", "vpn", "ns3", "mail2", "new",
	"mysql", "old", "www1", "www3", "api", "cdn", "api2", "beta", "shop", "stage",
	"staging", "app", "mobile", "secure", "demo", "portal", "server", "mx", "email",
	"cloud", "cdn1", "cdn2", "static", "img", "images", "media", "assets", "js", "css",
	"support", "help", "status", "monitor", "grafana", "kibana", "jenkins", "gitlab",
	"git", "svn", "jira", "confluence", "wiki", "docs", "developer", "dev1", "dev2",
	"test1", "test2", "uat", "qa", "preprod", "prod", "production", "backup", "db",
	"database", "sql", "redis", "mongo", "elastic", "search", "solr", "vpn1", "remote",
	"owa", "exchange", "lync", "sip", "lyncdiscover", "meet", "webconf", "conf",
	"sso", "auth", "login", "account", "accounts", "id", "identity", "oauth", "openid",
	"chat", "matrix", "element", "rocket", "mattermost", "discourse", "community",
	"drive", "storage", "files", "file", "share", "shared", "nextcloud", "owncloud",
	"voice", "voip", "pbx", "asterisk", "freepbx", "3cx", "voicegw", "sipgw",
	"notification", "notify", "push", "webhook", "hooks", "callback",
	"groupcall", "call", "video", "jitsi",
	"naga", "naga-ai", "ai", "ml", "data", "analytics", "bi", "report", "reports",
	"dashboard", "panel", "control", "manage", "management", "console", "backend",
	"frontend", "web", "www-dev", "www-test", "www-staging", "www-prod",
	"api-dev", "api-test", "api-staging", "api-prod", "api-v1", "api-v2", "v1", "v2",
	"old-api", "new-api", "internal", "intranet", "extranet", "partner", "partners",
	"client", "clients", "customer", "customers", "user", "users", "member", "members",
	"shop", "store", "cart", "order", "orders", "payment", "pay", "billing", "invoice",
	"crm", "erp", "hr", "hrm", "payroll", "recruit", "career", "careers", "job", "jobs",
	"blog", "news", "press", "media", "pr", "marketing", "ads", "ad", "track", "tracking",
	"cdn-assets", "static-assets", "img1", "img2", "image", "photo", "photos", "gallery",
	"video", "videos", "stream", "live", "tv", "radio", "podcast",
	"mail1", "mail2", "mail3", "smtp1", "smtp2", "mx1", "mx2", "mx3",
	"ns4", "ns5", "dns", "dns1", "dns2", "resolver",
	"ftp1", "ftp2", "sftp", "ssh", "repo", "repository", "code",
	"ci", "cd", "build", "deploy", "release", "pipeline", "runner",
	"log", "logs", "logging", "elk", "graylog", "splunk", "monitoring",
	"prometheus", "alert", "alerts", "uptime", "statuspage",
	"backup1", "backup2", "bak", "archive", "archives", "legacy",
	"testbed", "sandbox", "lab", "labs", "research", "experiment",
	"proxy", "gateway", "gw", "lb", "loadbalancer", "ha", "cluster",
	"k8s", "kubernetes", "docker", "swarm", "nomad", "consul", "vault",
	"redis1", "redis2", "memcached", "cache", "caching",
	"queue", "mq", "rabbit", "rabbitmq", "kafka", "nats", "pubsub",
	"search1", "elasticsearch", "opensearch", "solr1", "sphinx",
	"db1", "db2", "mysql1", "mysql2", "postgres", "postgresql", "mongo1", "mongodb",
	"oracle", "mssql", "sqlserver", "service", "services", "studio", "sentry", "monitor", "monitoring",
	"status", "health", "metrics", "trace", "tracing", "apm",
	"error", "errors", "observability", "cmp", "cmp-cms", "cms", "report-um",
}

// ====================== MAIN ======================

func main() {
	loadConfig()
	setupDirs()

	fmt.Println("[*] SubMonitor Pro (Python-power port)")
	fmt.Printf("[*] OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("[*] Domains: %v\n", cfg.Domains)

	for {
		runScan()
		if cfg.ScheduleHours <= 0 {
			break
		}
		fmt.Printf("\n[*] Wait %d hours for re-run...\n", cfg.ScheduleHours)
		time.Sleep(time.Duration(cfg.ScheduleHours) * time.Hour)
	}
}

// ====================== CONFIG ======================

func loadConfig() {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Println("[!] Can't find config file config.yaml")
		os.Exit(1)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		panic(err)
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "output"
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "data"
	}
	if cfg.ScheduleHours == 0 {
		cfg.ScheduleHours = 72
	}
}

func setupDirs() {
	dataDir = cfg.DataDir
	outputDir = cfg.OutputDir
	os.MkdirAll(toolsDir, 0755)
	os.MkdirAll(dataDir, 0755)
	os.MkdirAll(outputDir, 0755)
}

// ====================== SCAN ======================

func runScan() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("[*] Start scan...")
	fmt.Println(strings.Repeat("=", 60))

	var allResults []DomainResult

	for _, domain := range cfg.Domains {
		domain = strings.TrimSpace(strings.ToLower(domain))
		if domain == "" {
			continue
		}
		fmt.Printf("\n[+] Domain: %s\n", domain)

		result := scanDomain(domain)
		allResults = append(allResults, result)

		news := compareAndUpdate(domain, result)
		if len(news) > 0 {
			sendTelegram(domain, news)
		}
	}

	exportExcel(allResults)
}

func scanDomain(domain string) DomainResult {
	subs := make(map[string]bool)
	var mu sync.Mutex

	// 1. Passive sources (chạy song song)
	fmt.Println("  [*] Passive sources...")
	var wg sync.WaitGroup
	sources := []func(string) map[string]bool{
		fromCrtSh, fromHackerTarget, fromAlienVault,
		fromRapidDNS, fromCertSpotter, fromAnubis, fromThreatMiner,
	}

	for _, fn := range sources {
		wg.Add(1)
		go func(f func(string) map[string]bool) {
			defer wg.Done()
			found := f(domain)
			mu.Lock()
			for s := range found {
				subs[s] = true
			}
			mu.Unlock()
		}(fn)
	}
	wg.Wait()
	fmt.Printf("  [+] Passive: %d\n", len(subs))

	// 3. Brute-force
	fmt.Println("  [*] Brute-force...")
	for s := range bruteForce(domain) {
		subs[s] = true
	}

	// 4. Mutation
	fmt.Println("  [*] Mutation...")
	for s := range mutate(subs, domain) {
		subs[s] = true
	}

	// Resolve IP
	var records []SubdomainRecord
	now := time.Now()
	for sub := range subs {
		ip := resolveIP(sub)
		records = append(records, SubdomainRecord{
			Subdomain: sub,
			IP:        ip,
			FirstSeen: now,
			LastSeen:  now,
		})
	}

	fmt.Printf("  [+] Total : %d subdomain\n", len(records))
	return DomainResult{Domain: domain, Records: records, ScannedAt: now}
}

// ====================== PASSIVE SOURCES ======================

func fromCrtSh(domain string) map[string]bool {
	return fetchJSON(fmt.Sprintf("https://crt.sh/?q=%%25.%s&output=json", url.QueryEscape(domain)), domain, func(body []byte) []string {
		var data []map[string]interface{}
		json.Unmarshal(body, &data)
		var res []string
		for _, e := range data {
			if v, ok := e["name_value"].(string); ok {
				for _, line := range strings.Split(v, "\n") {
					res = append(res, line)
				}
			}
		}
		return res
	})
}

func fromHackerTarget(domain string) map[string]bool {
	body, err := httpGet("https://api.hackertarget.com/hostsearch/?q=" + url.QueryEscape(domain))
	result := make(map[string]bool)
	if err != nil {
		return result
	}
	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Split(line, ",")
		if len(parts) > 0 {
			if h := cleanHost(parts[0], domain); h != "" {
				result[h] = true
			}
		}
	}
	return result
}

func fromAlienVault(domain string) map[string]bool {
	return fetchJSON(fmt.Sprintf("https://otx.alienvault.com/api/v1/indicators/domain/%s/passive_dns", url.PathEscape(domain)), domain, func(body []byte) []string {
		var data struct {
			PassiveDNS []struct {
				Hostname string `json:"hostname"`
			} `json:"passive_dns"`
		}
		json.Unmarshal(body, &data)
		var res []string
		for _, r := range data.PassiveDNS {
			res = append(res, r.Hostname)
		}
		return res
	})
}

func fromRapidDNS(domain string) map[string]bool {
	body, err := httpGet("https://rapiddns.io/subdomain/" + url.PathEscape(domain) + "?full=1")
	result := make(map[string]bool)
	if err != nil {
		return result
	}
	// Get hostname contains domain
	for _, line := range strings.Split(string(body), "\n") {
		if strings.Contains(line, domain) {
			// get hostname
			fields := strings.Fields(line)
			for _, f := range fields {
				if h := cleanHost(f, domain); h != "" {
					result[h] = true
				}
			}
		}
	}
	return result
}

func fromCertSpotter(domain string) map[string]bool {
	return fetchJSON(fmt.Sprintf("https://api.certspotter.com/v1/issuances?domain=%s&include_subdomains=true&expand=dns_names", url.QueryEscape(domain)), domain, func(body []byte) []string {
		var data []struct {
			DNSNames []string `json:"dns_names"`
		}
		json.Unmarshal(body, &data)
		var res []string
		for _, e := range data {
			res = append(res, e.DNSNames...)
		}
		return res
	})
}

func fromAnubis(domain string) map[string]bool {
	return fetchJSON("https://jldc.me/anubis/subdomains/"+url.PathEscape(domain), domain, func(body []byte) []string {
		var list []string
		json.Unmarshal(body, &list)
		return list
	})
}

func fromThreatMiner(domain string) map[string]bool {
	return fetchJSON("https://api.threatminer.org/v2/domain.php?q="+url.QueryEscape(domain)+"&rt=5", domain, func(body []byte) []string {
		var data struct {
			Results []string `json:"results"`
		}
		json.Unmarshal(body, &data)
		return data.Results
	})
}

// ====================== BRUTE + MUTATION ======================

func bruteForce(domain string) map[string]bool {
	result := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, sub := range defaultWordlist {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			host := s + "." + domain
			if resolveIP(host) != "N/A" {
				mu.Lock()
				result[host] = true
				mu.Unlock()
			}
		}(sub)
	}
	wg.Wait()
	return result
}

func mutate(base map[string]bool, domain string) map[string]bool {
	result := make(map[string]bool)
	prefixes := make(map[string]bool)

	for s := range base {
		if strings.HasSuffix(s, "."+domain) {
			p := strings.TrimSuffix(s, "."+domain)
			if p != "" && !strings.Contains(p, ".") {
				prefixes[p] = true
			}
		}
	}

	patterns := []string{"%s", "%s1", "%s2", "%s-dev", "%s-test", "%s-staging", "dev-%s", "test-%s", "api-%s"}
	var mu sync.Mutex
	var wg sync.WaitGroup

	for p := range prefixes {
		for _, pat := range patterns {
			wg.Add(1)
			go func(prefix, pattern string) {
				defer wg.Done()
				cand := fmt.Sprintf(pattern, prefix) + "." + domain
				if resolveIP(cand) != "N/A" {
					mu.Lock()
					result[cand] = true
					mu.Unlock()
				}
			}(p, pat)
		}
	}
	wg.Wait()
	return result
}

// ====================== WORDLIST ======================
func loadWordlist() []string {
	if cfg.Wordlist == "" {
		return defaultWordlist
	}
	data, err := os.ReadFile(cfg.Wordlist)
	if err != nil {
		fmt.Printf("  [!] Can't read the wordlist, using defaultwordlist\n")
		return defaultWordlist
	}
	var list []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.ToLower(line))
		if line != "" && !strings.HasPrefix(line, "#") {
			list = append(list, line)
		}
	}
	if len(list) == 0 {
		return defaultWordlist
	}
	fmt.Printf("  [+] Loading wordlist: %d from\n", len(list))
	return list
}

// ====================== HELPERS ======================

func fetchJSON(api, domain string, parser func([]byte) []string) map[string]bool {
	result := make(map[string]bool)
	body, err := httpGet(api)
	if err != nil {
		return result
	}
	for _, h := range parser(body) {
		if clean := cleanHost(h, domain); clean != "" {
			result[clean] = true
		}
	}
	return result
}

func httpGet(api string) ([]byte, error) {
	client := &http.Client{Timeout: 18 * time.Second}
	req, _ := http.NewRequest("GET", api, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SubMonitor/3.0)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func cleanHost(host, domain string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimPrefix(host, "*.")
	host = strings.Trim(host, `."'`)

	// Allow only valid hostname characters
	for _, c := range host {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '.') {
			return ""
		}
	}

	// It must be the correct domain or its subdomain.
	if host == domain || strings.HasSuffix(host, "."+domain) {
		// Cannot begin with a period or contain two consecutive periods
		if strings.HasPrefix(host, ".") || strings.Contains(host, "..") {
			return ""
		}
		return host
	}
	return ""
}

func resolveIP(host string) string {
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return "N/A"
	}
	return ips[0].String()
}

// ====================== HISTORY + TELEGRAM + EXCEL ======================

func compareAndUpdate(domain string, current DomainResult) []SubdomainRecord {
	oldFile := filepath.Join(dataDir, domain+".json")
	var old DomainResult
	if data, err := os.ReadFile(oldFile); err == nil {
		json.Unmarshal(data, &old)
	}

	oldMap := make(map[string]SubdomainRecord)
	for _, r := range old.Records {
		oldMap[r.Subdomain] = r
	}

	var news []SubdomainRecord
	now := time.Now()
	for i := range current.Records {
		rec := &current.Records[i]
		if prev, ok := oldMap[rec.Subdomain]; ok {
			rec.FirstSeen = prev.FirstSeen
			rec.LastSeen = now
		} else {
			rec.FirstSeen = now
			rec.LastSeen = now
			news = append(news, *rec)
		}
	}
	data, _ := json.MarshalIndent(current, "", "  ")
	os.WriteFile(oldFile, data, 0644)
	return news
}

func sendTelegram(domain string, news []SubdomainRecord) {
	if cfg.Telegram.BotToken == "" || cfg.Telegram.ChatID == "" {
		return
	}
	msg := fmt.Sprintf("🔔 *New Subdomain* on `%s`\n\n", domain)
	for _, r := range news {
		msg += fmt.Sprintf("• `%s` → `%s`\n", r.Subdomain, r.IP)
	}
	msg += "\n" + time.Now().Format("2006-01-02 15:04:05")

	api := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.Telegram.BotToken)
	data := url.Values{}
	data.Set("chat_id", cfg.Telegram.ChatID)
	data.Set("text", msg)
	data.Set("parse_mode", "Markdown")
	http.PostForm(api, data)
	fmt.Printf("  [+] Telegram: %d subdomain mới\n", len(news))
}

func exportExcel(results []DomainResult) {
	f := excelize.NewFile()
	defer f.Close()

	for i, res := range results {
		sheet := res.Domain
		if len(sheet) > 31 {
			sheet = sheet[:31]
		}
		if i == 0 {
			f.SetSheetName("Sheet1", sheet)
		} else {
			f.NewSheet(sheet)
		}

		f.SetCellValue(sheet, "A1", "Subdomain")
		f.SetCellValue(sheet, "B1", "IP")
		f.SetCellValue(sheet, "C1", "First Seen")
		f.SetCellValue(sheet, "D1", "Last Seen")

		// Include with IP
		ipGroups := make(map[string][]SubdomainRecord)
		for _, r := range res.Records {
			ipGroups[r.IP] = append(ipGroups[r.IP], r)
		}

		row := 2
		for ip, recs := range ipGroups {
			for _, r := range recs {
				f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.Subdomain)
				f.SetCellValue(sheet, fmt.Sprintf("B%d", row), ip)
				f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r.FirstSeen.Format("2006-01-02 15:04"))
				f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r.LastSeen.Format("2006-01-02 15:04"))
				row++
			}
			row++
		}

		f.SetColWidth(sheet, "A", "A", 45)
		f.SetColWidth(sheet, "B", "B", 18)
		f.SetColWidth(sheet, "C", "C", 18)
		f.SetColWidth(sheet, "D", "D", 18)
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("subdomains_%s.xlsx", time.Now().Format("20060102_150405")))
	if err := f.SaveAs(filename); err != nil {
		fmt.Println("[!] Lỗi Excel:", err)
		return
	}
	fmt.Printf("\n[+] Report export to: %s\n", filename)
}
