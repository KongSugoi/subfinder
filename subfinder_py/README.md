# Tool subfinder

## 1. Install 

```
git clone https://github.com/KongSugoi/subfinder.git

pip install -r requirements.txt
```

## 2. Usage

```
python .\subfinder.py -h
usage: subfinder.py [-h] -d DOMAIN [-w WORDLIST] [-o OUTPUT] [--excel] [--no-brute] [--no-mutate] [--alive-only]
```
## 3. How to use
```python
# Find subdomain of 1 domain 

python subfinder.py -d DOMAIN

# example: python subfinder.py -d example.com

# Find subdomain alive of 1 domain 

python subfinder.py -d DOMAIN --alive-only
# example: python subfinder.py -d example.com --alive-only

# Export file to excel/csv

python subfinder.py -d DOMAIN --excel

## example: python subfinder.py -d example.com --excel
```
## 4. Source building

- [crt.sh](https://github.com/az7rb/crt.sh)
- [HackerTarget](https://hackertarget.com/find-dns-host-records/)
- [amass](https://github.com/owasp-amass/amass)

