#bin/bash
APP_NAME="srs-resolver"
BUILD_DIR="bin"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/srs-resolver"
LOG_DIR="/var/log"

echo "Installing srs-resolver..."

install -Dm 755 $BUILD_DIR/$APP_NAME $INSTALL_DIR/$APP_NAME
install -Dm 644 config/srs-resolver.conf $CONFIG_DIR/srs-resolver.conf
install -Dm 664 /dev/null $LOG_DIR/srs-resolver.log

echo "Registering service..."

install -Dm 664 systemd/srs-resolver.service /etc/systemd/system/srs-resolver.service
systemctl daemon-reexec
systemctl daemon-reload
systemctl enable srs-resolver.service
systemctl start srs-resolver.service

echo "Done."