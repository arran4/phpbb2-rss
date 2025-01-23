# README

## Overview
This application converts a PHPBB2 topic list page to a RSS feed. Ideally the "View posts from last 24 hours" page.

## Installation

### Prerequisites
- **Go** (if building from source)

#### Optional
- **Apache or Nginx** (for serving RSS files)

### Build and Install

#### Install only (System level)
Grab the latest binary here: https://github.com/arran4/phpbb2-rss/releases/

#### Install and build as user (User)
Install go 1.23+

Run `go install`:
```bash
go install github.com/arran4/phpbb2-rss/cmd/phpbb2rss@latest
```
This installs to `$HOME/go/bin` (typically; check with `go env`).

### Usage
#### CLI Mode
Generate RSS Feed:
```bash
phpbb2rss -output /var/www/localhost/htdocs/rss/phpbb2rss.xml  -url https://forums.$HOST.org/search.php?search_id=last
```

### Deployment

#### rc.d (Cron Job system level)
Add a cron job to run the script periodically:
1. Edit the root crontab:
   ```bash
   sudo crontab -e
   ```
2. Add the following line:
   ```bash
   */15 * * * * /usr/local/bin/phpbb2rss -output /var/www/localhost/htdocs/rss/phpbb2rss.xml  -url https://forums.$HOST.org/search.php?search_id=last
   ```

#### rc.d (Cron Job user level)
Add a cron job to run the script periodically:
1. Edit the user's crontab:
   ```bash
   crontab -e
   ```
2. Add the following line:
   ```bash
   */15 * * * * ~/go/bin/phpbb2rss -output ~/public_html/rss/phpbb2rss.xml -url https://forums.$HOST.org/search.php?search_id=last
   ```

#### systemd (as root)
1. Create a systemd service file at `/etc/systemd/system/phpbb2rss.service`:
```ini
[Unit]
Description=phpbb2 to RSS Feed Creator

[Service]
Type=oneshot
ExecStart=/usr/bin/phpbb2rss -output /var/www/localhost/htdocs/rss/phpbb2rss.xml
User=apache
Group=apache
```

2. Create a systemd timer file at `/etc/systemd/system/everyhour@.timer`:

```ini
[Unit]
Description=Monthly Timer for %i service

[Timer]
OnCalendar=*-*-* *:00:00
AccuracySec=1h
RandomizedDelaySec=1h
Persistent=true
Unit=%i.service

[Install]
WantedBy=default.target
```

3. Reload systemd and start the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable --now everyhour@phpbb2rss.timer
   ```

#### systemd (as user)
1. Create a systemd service file at `$HOME/.config/systemd/user/phpbb2rss.service`:
```ini
[Unit]
Description=phpbb2 to RSS Feed Creator

[Service]
Type=oneshot
ExecStart=%h/go/bin/phpbb2rss -output %h/public_html/rss/phpbb2rss.xml -url https://forums.$HOST.org/search.php?search_id=last
```

2. Create a systemd timer file at `$HOME/.config/systemd/user/everyhour@.timer`:

```ini
[Unit]
Description=Monthly Timer for %i service

[Timer]
OnCalendar=*-*-* *:00:00
AccuracySec=1h
RandomizedDelaySec=1h
Persistent=true
Unit=%i.service

[Install]
WantedBy=default.target
```

3. Reload systemd and start the service:
   ```bash
   systemctl --user daemon-reload && systemctl --user enable --now everyhour@phpbb2rss.timer
   ```

#### Apache VirtualHost Configuration
##### User

Refer to documentation for setting up public_html directories

##### Enjoy

http://localhost/~$USERNAME/rss/phpbb2rss.xml

##### System

Add the following configuration to your Apache setup (e.g., `/etc/httpd/conf.d/rss.conf`):
```apache
<VirtualHost *:80>
    ServerName example.com
    DocumentRoot /var/www/localhost/htdocs/rss
    <Directory "/var/www/localhost/htdocs/rss">
        Options Indexes FollowSymLinks
        AllowOverride None
        Require all granted
    </Directory>
</VirtualHost>
```

#### Nginx Configuration
##### User

Refer to documentation for setting up public_html directories

##### System

Add this to your Nginx server block:
```nginx
server {
    listen 80;
    server_name example.com;

    location /rss/ {
        root /var/www/localhost/htdocs;
        autoindex on;
    }
}
```
