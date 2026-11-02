#!/usr/bin/env python3
"""
Subdomain Finder Pro - Full Version
Xuất: TXT + CSV + Excel
"""

import argparse
import concurrent.futures
import csv
import re
import socket
import sys
from typing import Set, List
from urllib.parse import quote

import requests

# ================== CẤU HÌNH ==================
TIMEOUT = 12
MAX_WORKERS = 40
HEADERS = {
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
}

# Wordlist đã mở rộng (bao gồm service, studio, sentry)
COMMON_SUBS = [
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
    "oracle", "mssql", "sqlserver",
    # Các từ còn thiếu so với Pentest-Tools
    "service", "services", "studio", "sentry", "monitor", "monitoring",
    "status", "health", "metrics", "trace", "tracing", "apm",
    "error", "errors", "observability",
]

# Mutation patterns
MUTATION_PATTERNS = [
    "{}", "{}1", "{}2", "{}3", "{}01", "{}02",
    "{}-1", "{}-2", "{}-dev", "{}-test", "{}-staging", "{}-prod", "{}-old", "{}-new",
    "{}-api", "{}-app", "{}-web", "{}-admin", "{}-portal",
    "dev-{}", "test-{}", "staging-{}", "prod-{}", "old-{}", "new-{}",
    "api-{}", "app-{}", "web-{}", "admin-{}", "internal-{}",
    "{}dev", "{}test", "{}staging", "{}prod",
    "m-{}", "mobile-{}", "www-{}",
]


def resolve(host: str) -> bool:
    try:
        socket.setdefaulttimeout(2.5)
        socket.gethostbyname(host)
        return True
    except Exception:
        return False


def from_crtsh(domain: str) -> Set[str]:
    subs = set()
    url = f"https://crt.sh/?q=%25.{quote(domain)}&output=json"
    try:
        r = requests.get(url, headers=HEADERS, timeout=25)
        if r.status_code == 200:
            for entry in r.json():
                name = entry.get("name_value") or entry.get("common_name") or ""
                for line in name.splitlines():
                    line = line.strip().lower()
                    if line.startswith("*."):
                        line = line[2:]
                    if line.endswith(f".{domain}") or line == domain:
                        subs.add(line)
    except Exception as e:
        print(f"  [!] crt.sh: {e}")
    return subs


def from_hackertarget(domain: str) -> Set[str]:
    subs = set()
    url = f"https://api.hackertarget.com/hostsearch/?q={quote(domain)}"
    try:
        r = requests.get(url, headers=HEADERS, timeout=TIMEOUT)
        if r.status_code == 200 and not r.text.startswith("error"):
            for line in r.text.strip().splitlines():
                if "," in line:
                    host = line.split(",")[0].strip().lower()
                    if host.endswith(f".{domain}") or host == domain:
                        subs.add(host)
    except Exception as e:
        print(f"  [!] HackerTarget: {e}")
    return subs


def from_alienvault(domain: str) -> Set[str]:
    subs = set()
    url = f"https://otx.alienvault.com/api/v1/indicators/domain/{quote(domain)}/passive_dns"
    try:
        r = requests.get(url, headers=HEADERS, timeout=TIMEOUT)
        if r.status_code == 200:
            for record in r.json().get("passive_dns", []):
                host = record.get("hostname", "").lower()
                if host.endswith(f".{domain}") or host == domain:
                    subs.add(host)
    except Exception as e:
        print(f"  [!] AlienVault: {e}")
    return subs


def from_rapiddns(domain: str) -> Set[str]:
    subs = set()
    url = f"https://rapiddns.io/subdomain/{quote(domain)}?full=1"
    try:
        r = requests.get(url, headers=HEADERS, timeout=15)
        if r.status_code == 200:
            found = re.findall(rf'([\w\.-]+\.{re.escape(domain)})', r.text, re.I)
            for host in found:
                host = host.lower()
                if host.endswith(f".{domain}") or host == domain:
                    subs.add(host)
    except Exception as e:
        print(f"  [!] RapidDNS: {e}")
    return subs


def from_certspotter(domain: str) -> Set[str]:
    """CertSpotter - Certificate Transparency"""
    subs = set()
    url = f"https://api.certspotter.com/v1/issuances?domain={quote(domain)}&include_subdomains=true&expand=dns_names"
    try:
        r = requests.get(url, headers=HEADERS, timeout=TIMEOUT)
        if r.status_code == 200:
            for entry in r.json():
                for name in entry.get("dns_names", []):
                    name = name.lower().strip()
                    if name.startswith("*."):
                        name = name[2:]
                    if name.endswith(f".{domain}") or name == domain:
                        subs.add(name)
    except Exception as e:
        print(f"  [!] CertSpotter: {e}")
    return subs


def from_anubis(domain: str) -> Set[str]:
    """AnubisDB"""
    subs = set()
    url = f"https://jldc.me/anubis/subdomains/{quote(domain)}"
    try:
        r = requests.get(url, headers=HEADERS, timeout=TIMEOUT)
        if r.status_code == 200:
            data = r.json()
            for host in data:
                host = host.lower().strip()
                if host.endswith(f".{domain}") or host == domain:
                    subs.add(host)
    except Exception as e:
        print(f"  [!] AnubisDB: {e}")
    return subs


def from_threatminer(domain: str) -> Set[str]:
    """ThreatMiner"""
    subs = set()
    url = f"https://api.threatminer.org/v2/domain.php?q={quote(domain)}&rt=5"
    try:
        r = requests.get(url, headers=HEADERS, timeout=TIMEOUT)
        if r.status_code == 200:
            data = r.json()
            if data.get("status_code") == "200":
                for host in data.get("results", []):
                    host = host.lower().strip()
                    if host.endswith(f".{domain}") or host == domain:
                        subs.add(host)
    except Exception as e:
        print(f"  [!] ThreatMiner: {e}")
    return subs


def brute_force(domain: str, wordlist: List[str]) -> Set[str]:
    found = set()
    print(f"  [*] Brute-force với {len(wordlist)} từ khóa...")

    def check(sub):
        host = f"{sub}.{domain}".lower()
        if resolve(host):
            return host
        return None

    with concurrent.futures.ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
        futures = [executor.submit(check, w) for w in wordlist]
        for future in concurrent.futures.as_completed(futures):
            result = future.result()
            if result:
                found.add(result)
                print(f"    [+] {result}")
    return found


def mutate(base_subs: Set[str], domain: str) -> Set[str]:
    mutated = set()
    prefixes = set()

    for s in base_subs:
        if s.endswith(f".{domain}"):
            prefix = s[: -(len(domain) + 1)]
            if prefix and "." not in prefix and len(prefix) < 30:
                prefixes.add(prefix)

    print(f"  [*] Mutation từ {len(prefixes)} prefix...")

    candidates = set()
    for base in list(prefixes)[:40]:
        for pattern in MUTATION_PATTERNS:
            candidates.add(pattern.format(base))

    def check(cand):
        host = f"{cand}.{domain}".lower()
        if resolve(host):
            return host
        return None

    with concurrent.futures.ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
        futures = [executor.submit(check, c) for c in candidates]
        for future in concurrent.futures.as_completed(futures):
            result = future.result()
            if result:
                mutated.add(result)
                print(f"    [+] mutation: {result}")
    return mutated


def clean(subs: Set[str], domain: str) -> List[str]:
    domain = domain.lower()
    result = set()
    for s in subs:
        s = s.lower().strip(".").strip()
        if s and (s == domain or s.endswith(f".{domain}")):
            if re.fullmatch(r"[a-z0-9\.\-]+", s):
                result.add(s)
    return sorted(result)


def main():
    parser = argparse.ArgumentParser(description="Subdomain Finder Pro - Full Version")
    parser.add_argument("-d", "--domain", required=True, help="Domain mục tiêu")
    parser.add_argument("-w", "--wordlist", help="File wordlist tùy chỉnh")
    parser.add_argument("-o", "--output", help="Tên file kết quả (không cần đuôi)")
    parser.add_argument("--excel", action="store_true", help="Xuất thêm file Excel")
    parser.add_argument("--no-brute", action="store_true", help="Bỏ brute-force")
    parser.add_argument("--no-mutate", action="store_true", help="Bỏ mutation")
    parser.add_argument("--alive-only", action="store_true", help="Chỉ giữ subdomain đang sống")
    args = parser.parse_args()

    domain = args.domain.lower().strip()
    if not re.fullmatch(r"[a-z0-9\.\-]+", domain):
        print("Domain không hợp lệ")
        sys.exit(1)

    print(f"\n[*] Subdomain Finder Pro cho: {domain}")
    print("=" * 60)

    all_subs: Set[str] = set()

    # 1. Passive sources
    print("\n[1] Passive Sources...")
    with concurrent.futures.ThreadPoolExecutor(max_workers=7) as executor:
        futures = {
            executor.submit(from_crtsh, domain): "crt.sh",
            executor.submit(from_hackertarget, domain): "HackerTarget",
            executor.submit(from_alienvault, domain): "AlienVault",
            executor.submit(from_rapiddns, domain): "RapidDNS",
            executor.submit(from_certspotter, domain): "CertSpotter",
            executor.submit(from_anubis, domain): "AnubisDB",
            executor.submit(from_threatminer, domain): "ThreatMiner",
        }
        for future in concurrent.futures.as_completed(futures):
            name = futures[future]
            try:
                result = future.result()
                print(f"  [+] {name:12} → {len(result):4}")
                all_subs.update(result)
            except Exception as e:
                print(f"  [!] {name}: {e}")

    # 2. Brute-force
    if not args.no_brute:
        print("\n[2] DNS Brute-force...")
        wordlist = COMMON_SUBS.copy()
        if args.wordlist:
            try:
                with open(args.wordlist, encoding="utf-8", errors="ignore") as f:
                    custom = [line.strip().lower() for line in f if line.strip()]
                    wordlist = list(set(wordlist + custom))
                    print(f"  [*] Đã thêm {len(custom)} từ wordlist tùy chỉnh")
            except Exception as e:
                print(f"  [!] Lỗi đọc wordlist: {e}")

        brute_subs = brute_force(domain, wordlist)
        all_subs.update(brute_subs)

    # 3. Mutation
    if not args.no_mutate and all_subs:
        print("\n[3] Word Mutation...")
        mut_subs = mutate(all_subs, domain)
        all_subs.update(mut_subs)

    # 4. Clean
    print("\n[4] Cleaning...")
    final = clean(all_subs, domain)

    if args.alive_only:
        print("  [*] Lọc subdomain đang sống...")
        alive = []
        with concurrent.futures.ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
            results = list(executor.map(lambda h: (h, resolve(h)), final))
            for host, is_alive in results:
                if is_alive:
                    alive.append(host)
        final = sorted(alive)

    print("=" * 60)
    print(f"[*] Tổng subdomain tìm được: {len(final)}")

    # ========== XUẤT FILE ==========
    base_name = args.output or domain

    # TXT
    txt_file = f"{base_name}_subs.txt"
    with open(txt_file, "w", encoding="utf-8") as f:
        f.write("\n".join(final) + "\n")
    print(f"[+] TXT   → {txt_file}")

    # CSV
    csv_file = f"{base_name}_subs.csv"
    with open(csv_file, "w", encoding="utf-8-sig", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(["Subdomain"])
        for sub in final:
            writer.writerow([sub])
    print(f"[+] CSV   → {csv_file}")

    # Excel
    if args.excel:
        try:
            from openpyxl import Workbook
            excel_file = f"{base_name}_subs.xlsx"
            wb = Workbook()
            ws = wb.active
            ws.title = "Subdomains"
            ws.append(["Subdomain"])
            for sub in final:
                ws.append([sub])
            ws.column_dimensions['A'].width = 55
            wb.save(excel_file)
            print(f"[+] Excel → {excel_file}")
        except ImportError:
            print("[!] Chưa cài openpyxl → chạy: pip install openpyxl")

    if final:
        print("\nMột số kết quả:")
        for s in final[:25]:
            print(f"  • {s}")
        if len(final) > 25:
            print(f"  ... và {len(final)-25} subdomain khác")


if __name__ == "__main__":
    main()