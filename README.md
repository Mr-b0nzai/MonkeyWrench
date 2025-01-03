# 🐵🔧 Monkeywrench: The 403 Bypass Tool

## 🖊️ Description
Monkeywrench is a powerful Go-based tool aimed at bypassing HTTP 403 Forbidden responses. It offers a flexible approach to URL testing and header manipulation, making it an essential addition to a penetration tester's toolkit.

## 📦 Installation
The easies way to install MonkeyWrench is via `go install`
```
$ go install github.com/Mr-b0nzai/MonkeyWrench
```
To build the tool from source, ensure you have Go installed:
```
$ git clone https://github.com/yourusername/monkeywrench.git
$ cd monkeywrench
$ go build -o monkeywrench
```
## 📜 Usage
Run Monkeywrench using the following syntax:
```./monkeywrench --mode=<mode> [options]```

## 🛡️ Key Features
- Mode Selection: Choose between full and headers.
- Custom Headers: Specify additional headers for requests.
- Input Flexibility: Read URLs from a file or stdin.
- Burp-Compatible Requests: Easily copy requests to Burp Suite.
- YAML Response Output: Optionally format responses in YAML.

## 🔧 Options
| Flag       | Description                                                         |
|------------|---------------------------------------------------------------------|
| --mode     | Execution mode: `full` or `headers` (required).                     |
| --file     | File containing URLs (optional).                                    |
| --requests | Print HTTP requests in Burp Suite style (optional).                 |
| --yaml     | Enable YAML output for responses (optional).                        |
| --headers  | Custom headers in `Key: Value` format (optional).                   |

## 🛠️ Example
```./monkeywrench --mode=headers --file=urls.txt --headers="Authorization: Bearer token"```
### 🤝 Contributions
We welcome contributions to enhance the tool. Feel free to fork the repository and submit a pull request.

### 📄 License
Monkeywrench is licensed under the MIT License. See LICENSE for more information.

