
# SOCKS5 Proxy Setup (Dante)

Скрипт устанавливает и настраивает **SOCKS5 прокси** на сервере с использованием `dante-server`.

Особенности:

- SOCKS5 сервер
- порт **50000**
- **IP whitelist**
- автоматическое определение сетевого интерфейса
- **полная перезапись `/etc/danted.conf`**
- автозапуск через `systemd`

---

# Требования

- Ubuntu / Debian
- root доступ
- установлен `apt`

---

# Файлы

```

.
├── install_socks5.sh
└── whitelist.txt

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

# Установка

Необходимо скачать **два файла**:

### install.sh

```bash
wget https://raw.githubusercontent.com/Fox216540/socks5-proxy/main/install.sh -O install.sh
```
### whitelist.txt
```bash
wget https://raw.githubusercontent.com/Fox216540/socks5-proxy/main/whitelist.txt -O whitelist.txt
```

Сделать скрипт исполняемым:

```bash
chmod +x install_socks5.sh
````

Запустить установку:

```bash
sudo ./install_socks5.sh
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

# Изменение whitelist

1. Отредактировать `whitelist.txt`
2. Перезапустить скрипт

```bash
sudo ./install_socks5.sh
```

Конфигурация будет **пересоздана**.

---

# Логи

Посмотреть логи сервиса:

```bash
journalctl -u danted -f
```

---

# Удаление прокси

```bash
apt remove dante-server
```

или отключить сервис:

```bash
systemctl stop danted
systemctl disable danted
```

