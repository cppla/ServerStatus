#!/usr/bin/env bash
# ServerStatus 客户端一键安装脚本（systemd 方式）
# 自动下载 status-client.service 与 client-linux.py，注册 systemd 服务并启动。
# 用法: bash install.sh SERVER=服务器地址 [PORT=端口] [USER=用户名] [PASSWORD=密码]
set -euo pipefail

github_prefix="https://raw.githubusercontent.com/cppla/ServerStatus/master"

client_file="/usr/local/ServerStatus/clients/client-linux.py"
client_env="/usr/local/ServerStatus/clients/config.env"
client_service="/usr/lib/systemd/system/status-client.service"
client_override="/etc/systemd/system/status-client.service.d/override.conf"

SERVER=""
PORT="35601"
USER=""
PASSWORD=""

for arg in "$@"; do
  case "${arg}" in
    SERVER=*) SERVER="${arg#SERVER=}" ;;
    PORT=*) PORT="${arg#PORT=}" ;;
    USER=*) USER="${arg#USER=}" ;;
    PASSWORD=*) PASSWORD="${arg#PASSWORD=}" ;;
    *) echo "错误: 未知参数 ${arg}" >&2; exit 1 ;;
  esac
done

if [[ -z "${SERVER}" ]]; then
  echo "错误: 缺少 SERVER 参数，用法: bash install.sh SERVER=服务器地址 [PORT=端口] [USER=用户名] [PASSWORD=密码]" >&2
  exit 1
fi

if [[ -z "${USER}" ]]; then
  echo "警告: USER 为空，客户端可能无法上报，请用 USER=用户名 指定" >&2
fi

if [[ "$(id -u)" -ne 0 ]]; then
  if command -v sudo >/dev/null 2>&1; then
    exec sudo bash "$0" "$@"
  fi
  echo "错误: 请使用 root 权限运行（如: sudo bash install.sh ...）" >&2
  exit 1
fi

command -v wget >/dev/null 2>&1 || { echo "错误: 未找到 wget，请先安装 wget" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "错误: 未找到 python3，请先安装 python3" >&2; exit 1; }
command -v systemctl >/dev/null 2>&1 || { echo "错误: 未找到 systemctl，当前系统不支持 systemd" >&2; exit 1; }

mkdir -p "$(dirname "${client_file}")"
wget -qN --no-check-certificate "${github_prefix}/clients/client-linux.py" -O "${client_file}"
wget -qN --no-check-certificate "${github_prefix}/service/status-client.service" -O "${client_service}"
chmod +x "${client_file}"

printf 'SERVER=%s\nPORT=%s\nUSER=%s\nPASSWORD=%s\n' "${SERVER}" "${PORT}" "${USER}" "${PASSWORD}" > "${client_env}"

mkdir -p "$(dirname "${client_override}")"
cat > "${client_override}" <<'EOF'
[Service]
EnvironmentFile=/usr/local/ServerStatus/clients/config.env
EOF

systemctl daemon-reload
systemctl enable status-client >/dev/null 2>&1 || true
if ! systemctl restart status-client; then
  echo "错误: status-client 启动失败，最近日志:" >&2
  journalctl -u status-client -n 20 --no-pager >&2 || true
  exit 1
fi

echo "ServerStatus 客户端安装完成:"
echo "  SERVER: ${SERVER}"
echo "  PORT: ${PORT}"
echo "  USER: ${USER}"
echo "  配置: ${client_env}"
echo "  查看状态: systemctl status status-client"
echo "  查看日志: journalctl -u status-client -f"
