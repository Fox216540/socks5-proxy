# SOCKS5 Proxy Setup (Dante + Reverse Mobile Proxy)

Скрипт устанавливает и настраивает **SOCKS5 прокси** на сервере с использованием `dante-server`.

Дополнительно поддерживается **Reverse Mobile SOCKS**, позволяющий использовать **мобильный интернет телефона как прокси на VPS**.

---

# Возможности

### Dante SOCKS5

- SOCKS5 сервер
- порт **50000**
- **IP whitelist**
- автоматическое определение сетевого интерфейса
- полная перезапись `/etc/danted.conf`
- автозапуск через `systemd`

### Reverse Mobile Proxy

- мобильный интернет телефона как прокси
- reverse tunnel
- локальный SOCKS на VPS
- поддержка кастомных портов

---

# Требования

- Ubuntu / Debian
- root доступ
- установлен `apt`

---

# Файлы

```

.
├── install.sh
├── whitelist.txt
└── reverse.txt

```

---

# Формат whitelist.txt

Файл содержит список IP адресов, которым разрешён доступ к прокси.

```

1.2.3.4
5.6.7.8
9.10.11.12

````

Каждый IP указывается **с новой строки**.

---

# Установка SOCKS5 (Dante)

Скачать файлы:

### install.sh

```bash
wget https://raw.githubusercontent.com/Fox216540/socks5-proxy/main/install.sh -O install.sh
````

### reverse.txt

```bash
wget https://raw.githubusercontent.com/Fox216540/socks5-proxy/main/reverse.txt -O reverse.txt
```

### whitelist.txt

```bash
wget https://raw.githubusercontent.com/Fox216540/socks5-proxy/main/whitelist.txt -O whitelist.txt
```

Сделать скрипт исполняемым:

```bash
chmod +x install.sh
```

Запустить установку:

```bash
sudo ./install.sh
```

Скрипт:

1. устанавливает `dante-server`
2. определяет сетевой интерфейс
3. генерирует конфигурацию `/etc/danted.conf`
4. применяет whitelist
5. перезапускает сервис

---

# Подключение к прокси

```
socks5://SERVER_IP:50000
```

Пример:

```bash
curl --socks5 SERVER_IP:50000 https://ipinfo.io
```

---

# Проверка работы

Проверить что порт слушается:

```bash
ss -lntp | grep 50000
```

Проверить сервис:

```bash
systemctl status danted
```

Проверить соединение:

```bash
curl --socks5 127.0.0.1:50000 https://ipinfo.io
```

---

# Reverse Mobile SOCKS (через телефон)

Позволяет использовать **мобильный интернет телефона как SOCKS5 прокси на сервере**.

Архитектура:

```
Application on VPS
        │
   127.0.0.1:5010
        │
   Reverse tunnel
        │
   VPS:50003
        │
     Phone
        │
Mobile Internet
```

Телефон подключается к серверу и создаёт **reverse tunnel**.

После подключения на VPS появляется локальный SOCKS5.

---

# Установка Reverse сервера (VPS)

Скачать бинарник:

```bash
wget https://github.com/Fox216540/socks5-proxy/releases/latest/download/proxy-server-linux-amd64
```

Сделать исполняемым:

```bash
chmod +x proxy-server-linux-amd64
```

Запустить сервер:

```bash
./proxy-server-linux-amd64 50003 5010
```

где

```
50003 → порт подключения телефона
5010 → локальный SOCKS5 порт
```

После запуска:

```
SOCKS5 → 127.0.0.1:5010
reverse tunnel → :50003
```

---

# Установка клиента на телефон (Termux)

Скачать бинарник:

```bash
wget https://github.com/Fox216540/socks5-proxy/releases/latest/download/mobile-client-android-arm64
```

Сделать исполняемым:

```bash
chmod +x mobile-client-android-arm64
```

Запустить:

```bash
./mobile-client-android-arm64 SERVER_IP:50003
```

Пример:

```bash
./mobile-client-android-arm64 217.154.97.70:50003
```

После подключения сервер покажет:

```
mobile client connected
```

# Установка клиента на телефон (сборка в Termux)
## 1. Установить зависимости
```
pkg update
pkg install golang wget
```
## 2. Создать папку build
```
mkdir build
```
```
cd build
```
## 3. Скачать исходник клиента
```
wget https://raw.githubusercontent.com/Fox216540/socks5-proxy/main/build/mobile-client.go
```
### Если wget не работает:
```
curl -L -O https://raw.githubusercontent.com/Fox216540/socks5-proxy/main/build/mobile-client.go
```
## 4. Создать go.mod
### Инициализировать Go-модуль:
```
go mod init mobile-client
```
### Это создаст файл:
```
go.mod
```
## 5. Установить зависимости
```
go mod tidy
```
### Go автоматически скачает все необходимые зависимости.
## 6. Собрать клиент
```
go build -o mobile-client
```
## 7. Сделать исполняемым
```
chmod +x mobile-client
```
## 8. Запуск
```
./mobile-client SERVER_IP:50003
```
Пример:
```
./mobile-client 217.154.97.70:50003
```
---

# Проверка Mobile Proxy

На VPS:

```bash
curl --socks5 127.0.0.1:5010 https://api.ipify.org
```

Если всё работает — вернётся **IP мобильного оператора телефона**.

---

# Использование прокси

Пример:

```bash
curl --socks5 127.0.0.1:5010 https://ipinfo.io
```

или

```
socks5://127.0.0.1:5010
```

---

# Изменение портов

Reverse сервер позволяет указать порты при запуске:

```bash
./proxy-server-linux-amd64 REVERSE_PORT SOCKS_PORT
```

Пример:

```bash
./proxy-server-linux-amd64 50005 5015
```

---

# Логи

Dante:

```bash
journalctl -u danted -f
```

Reverse сервер:

```
mobile client connected
SOCKS5 listening on 127.0.0.1:5010
```

---

# Изменение whitelist

1. Отредактировать `whitelist.txt`
2. Перезапустить скрипт

```bash
sudo ./install.sh
```

Конфигурация будет **пересоздана**.

---

# Удаление прокси

Удалить Dante:

```bash
apt remove dante-server
```

Отключить сервис:

```bash
systemctl stop danted
systemctl disable danted
```

Остановить reverse сервер:

```
Ctrl + C
```

