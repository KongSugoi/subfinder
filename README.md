# SubMonitor

**SubMonitor** is an automated subdomain discovery tool written in Go.  
It combines multiple passive OSINT sources with DNS brute-force and mutation techniques to find subdomains, resolve their IP addresses, track first/last seen dates, and export results to Excel.

---

## Features

- **Multiple passive sources**:
  - crt.sh
  - HackerTarget
  - AlienVault OTX
  - RapidDNS
  - CertSpotter
  - AnubisDB
  - ThreatMiner

- DNS Brute-force using a built-in wordlist (or custom wordlist)
- Subdomain mutation / permutation
- Multi-domain support (each domain gets its own Excel sheet)
- IP resolution for every discovered subdomain
- Group results by IP (helps identify shared backends)
- Historical tracking (First Seen / Last Seen)
- Detect new subdomains compared to previous scans
- Telegram notification when new subdomains are found
- Scheduled scanning (default: every 72 hours)
- Export results to Excel (`.xlsx`)

---

## Requirements

- Go 1.21 or higher
- Internet connection

### Install dependencies

```bash
git clone https://github.com/KongSugoi/subfinder.git
cd submonitor
go mod tidy
```

---

## Configuration

Create a file named `config.yaml` in the same directory:

```yaml
domains:
  - example.com
  - another-domain.com
  - unitel.com.la

telegram:
  bot_token: "YOUR_TELEGRAM_BOT_TOKEN"
  chat_id: "YOUR_CHAT_ID"

schedule_hours: 72          # Set to 0 to run only once
output_dir: "output"
data_dir: "data"
wordlist: ""                # Leave empty to use built-in wordlist
                            # or provide path: "wordlist.txt"
```

### Telegram Setup (Optional)

1. Create a bot via [@BotFather](https://t.me/BotFather)
2. Get the bot token
3. Send a message to the bot
4. Visit: `https://api.telegram.org/bot<YourBotToken>/getUpdates`
5. Copy the `chat.id` value

---

## Usage

### Run once

```bash
go run main.go
```

### Build executable

```bash
# Linux / macOS
go build -o submonitor main.go

# Windows
go build -o submonitor.exe main.go
```

Then run:

```bash
./submonitor
# or
submonitor.exe
```

---

## Output

- Results are saved in the `output/` directory as Excel files:
  ```
  output/subdomains_YYYYMMDD_HHMMSS.xlsx
  ```
- Each domain has its own sheet
- Columns:
  - Subdomain
  - IP
  - First Seen
  - Last Seen

- Historical data is stored in the `data/` folder as JSON files (one per domain)

---

## Notes

- The tool does **not** require any external binaries (subfinder, amass, etc.)
- It relies only on public APIs + DNS resolution + wordlist
- Some sources may rate-limit or temporarily fail — the tool continues with available sources
- For better results, you can provide a larger custom wordlist via the `wordlist` option in `config.yaml`

---

## License

This project is for educational and authorized security testing purposes only.  
Always ensure you have permission before scanning any domain.
