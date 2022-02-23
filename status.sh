#!/usr/bin/env bash
PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin:~/bin
export PATH

sh_ver="1.0.0"

filepath=$(
  cd "$(dirname "$0")" || exit
  pwd
)
file_1=$(echo -e "${filepath}" | awk -F "$0" '{print $1}')
file="/usr/local/ServerStatus"
web_file="/usr/local/ServerStatus/web"
server_file="/usr/local/ServerStatus/server"
server_conf="/usr/local/ServerStatus/server/config.json"
server_conf_1="/usr/local/ServerStatus/server/config.conf"
plugin_file="/usr/local/ServerStatus/plugin"
client_file="/usr/local/ServerStatus/clients"
service="/usr/lib/systemd/system"
jq_file="${file}/jq"
[[ ! -e ${jq_file} ]] && jq_file="/usr/bin/jq"

github_prefix="https://raw.githubusercontent.com/cppla/ServerStatus/master"

NAME="ServerStatus"
Green_font_prefix="\033[32m" && Red_font_prefix="\033[31m" && Red_background_prefix="\033[41;37m" && Font_color_suffix="\033[0m"
Info="${Green_font_prefix}[信息]${Font_color_suffix}"
Error="${Red_font_prefix}[错误]${Font_color_suffix}"
Tip="${Green_font_prefix}[注意]${Font_color_suffix}"

#检查系统
check_sys() {
  if [[ -f /etc/redhat-release ]]; then
    release="centos"
  elif grep -q -E -i "debian|ubuntu" /etc/issue; then
    release="debian"
  elif grep -q -E -i "centonetstat -tunlps|red hat|redhat" /etc/issue; then
    release="centos"
  elif grep -q -E -i "Arch|Manjaro" /etc/issue; then
    release="archlinux"
  elif grep -q -E -i "debian|ubuntu" /proc/version; then
    release="debian"
  elif grep -q -E -i "centos|red hat|redhat" /proc/version; then
    release="centos"
  else
    echo -e "ServerStatus 暂不支持该Linux发行版"
  fi
  bit=$(uname -m)
}

check_installed_server_status() {
  [[ ! -e "${server_file}/sergate" ]] && echo -e "${Error} $NAME 服务端没有安装，请检查 !" && exit 1
}

check_installed_client_status() {
  [[ ! -e "${client_file}/client-linux.py" ]] && echo -e "${Error} $NAME 客户端没有安装，请检查 !" && exit 1
}

Download_Server_Status_server() {
  cd "/tmp" || exit 1
  wget -N --no-check-certificate https://github.com/cppla/ServerStatus/archive/refs/heads/master.zip
    [[ ! -e "master.zip" ]] && echo -e "${Error} ServerStatus 服务端下载失败 !" && exit 1
  unzip master.zip
  rm -rf master.zip
  [[ ! -d "/tmp/ServerStatus-master" ]] && echo -e "${Error} ServerStatus 服务端解压失败 !" && exit 1
  cd "/tmp/ServerStatus-master/server" || exit 1
  make
  [[ ! -e "sergate" ]] && echo -e "${Error} ServerStatus 服务端编译失败 !" && cd "${file_1}" && rm -rf "/tmp//ServerStatus-master" && exit 1
  cd "${file_1}" || exit 1
  mkdir -p "${server_file}"
  mv "/tmp/ServerStatus-master/server" "${file}"
  mv "/tmp/ServerStatus-master/web" "${file}"
  mv "/tmp/ServerStatus-master/plugin" "${file}"
  rm -rf "/tmp/ServerStatus-master"
  if [[ ! -e "${server_file}/sergate" ]]; then
    echo -e "${Error} ServerStatus 服务端移动重命名失败 !"
    [[ -e "${server_file}/sergate1" ]] && mv "${server_file}/sergate1" "${server_file}/sergate"
    exit 1
  else
    [[ -e "${server_file}/sergate1" ]] && rm -rf "${server_file}/sergate1"
  fi
}

Download_Server_Status_client() {
mkdir -p "${client_file}"
wget -N --no-check-certificate "${github_prefix}/clients/client-linux.py"  -P "${client_file}"
}

Download_Server_Status_Service() {
  mode=$1
  [[ -z ${mode} ]] && mode="server"
  local service_note="服务端"
  [[ ${mode} == "client" ]] && service_note="客户端"
    wget --no-check-certificate "${github_prefix}/service/status-${mode}.service" -O "${service}/status-${mode}.service" ||
      {
        echo -e "${Error} $NAME ${service_note}服务管理脚本下载失败 !"
        exit 1
      }
    systemctl enable "status-${mode}.service"
  echo -e "${Info} $NAME ${service_note}服务管理脚本下载完成 !"
}

Service_Server_Status_server() {
  Download_Server_Status_Service "server"
}

Service_Server_Status_client() {
  Download_Server_Status_Service "client"
}

Installation_dependency() {
  mode=$1
  if [[ ${release} == "centos" ]]; then
    yum makecache
    yum -y install unzip
    yum -y install python3 >/dev/null 2>&1 || yum -y install python
    [[ ${mode} == "server" ]] && yum -y groupinstall "Development Tools"
  elif [[ ${release} == "debian" ]]; then
    apt -y update
    apt -y install unzip
    apt -y install python3 >/dev/null 2>&1 || apt -y install python
    [[ ${mode} == "server" ]] && apt -y install build-essential
  elif [[ ${release} == "archlinux" ]]; then
    pacman -Sy python python-pip unzip --noconfirm
    [[ ${mode} == "server" ]] && pacman -Sy base-devel --noconfirm
  fi
  [[ ! -e /usr/bin/python ]] && ln -s /usr/bin/python3 /usr/bin/python
}

Write_server_config() {
  cat >${server_conf} <<-EOF
{
    "servers": [
        {
            "username": "s01",
            "password": "password",
            "name": "vps-1",
            "type": "KVM",
            "host": "azure",
            "location": "Hong Kong",
            "monthstart": 1
        }
    ]
}     
EOF
}

Write_server_config_conf() {
  cat >${server_conf_1} <<-EOF
PORT = ${server_port_s}
EOF
}

Read_config_client() {
  client_text="$(sed 's/\"//g;s/,//g;s/ //g' "${client_file}/client-linux.py") "
  client_server="$(echo -e "${client_text}" | grep "SERVER=" | awk -F "=" '{print $2;exit}')"
  client_port="$(echo -e "${client_text}" | grep "PORT=" | awk -F "=" '{print $2;exit}')"
  client_user="$(echo -e "${client_text}" | grep "USER=" | awk -F "=" '{print $2;exit}')"
  client_password="$(echo -e "${client_text}" | grep "PASSWORD=" | awk -F "=" '{print $2;exit}')"
}

Read_config_server() {
  if [[ ! -e "${server_conf_1}" ]]; then
    server_port_s="35601"
    Write_server_config_conf
    server_port="35601"
  else
    server_port="$(grep "PORT = " ${server_conf_1} | awk '{print $3}')"
  fi
}

Set_server() {
  mode=$1
  [[ -z ${mode} ]] && mode="server"
  if [[ ${mode} == "server" ]]; then
    echo -e "请输入 $NAME 服务端中网站要设置的 域名[server]
默认为本机IP为域名，例如输入: toyoo.pw ，如果要使用本机IP，请留空直接回车"
    read -erp "(默认: 本机IP):" server_s
    [[ -z "$server_s" ]] && server_s=""
  else
    echo -e "请输入 $NAME 服务端的 IP/域名[server]，请注意，如果你的域名使用了CDN，请直接填写IP"
    read -erp "(默认: 127.0.0.1):" server_s
    [[ -z "$server_s" ]] && server_s="127.0.0.1"
  fi

  echo && echo "	================================================"
  echo -e "	IP/域名[server]: ${Red_background_prefix} ${server_s} ${Font_color_suffix}"
  echo "	================================================" && echo
}

Set_server_http_port() {
  while true; do
    echo -e "请输入 $NAME 服务端中网站要设置的 域名/IP的端口[1-65535]（如果是域名的话，一般用 80 端口）"
    read -erp "(默认: 8888):" server_http_port_s
    [[ -z "$server_http_port_s" ]] && server_http_port_s="8888"
    if [[ "$server_http_port_s" =~ ^[0-9]*$ ]]; then
      if [[ ${server_http_port_s} -ge 1 ]] && [[ ${server_http_port_s} -le 65535 ]]; then
        echo && echo "	================================================"
        echo -e "	端口: ${Red_background_prefix} ${server_http_port_s} ${Font_color_suffix}"
        echo "	================================================" && echo
        break
      else
        echo "输入错误, 请输入正确的端口。"
      fi
    else
      echo "输入错误, 请输入正确的端口。"
    fi
  done
}

Set_server_port() {
  while true; do
    echo -e "请输入 $NAME 服务端监听的端口[1-65535]（用于服务端接收客户端消息的端口，客户端要填写这个端口）"
    read -erp "(默认: 35601):" server_port_s
    [[ -z "$server_port_s" ]] && server_port_s="35601"
    if [[ "$server_port_s" =~ ^[0-9]*$ ]]; then
      if [[ ${server_port_s} -ge 1 ]] && [[ ${server_port_s} -le 65535 ]]; then
        echo && echo "	================================================"
        echo -e "	端口: ${Red_background_prefix} ${server_port_s} ${Font_color_suffix}"
        echo "	================================================" && echo
        break
      else
        echo "输入错误, 请输入正确的端口。"
      fi
    else
      echo "输入错误, 请输入正确的端口。"
    fi
  done
}

Set_username() {
  mode=$1
  [[ -z ${mode} ]] && mode="server"
  if [[ ${mode} == "server" ]]; then
    echo -e "请输入 $NAME 服务端要设置的用户名[username]（字母/数字，不可与其他账号重复）"
  else
    echo -e "请输入 $NAME 服务端中对应配置的用户名[username]（字母/数字，不可与其他账号重复）"
  fi
  read -erp "(默认: 取消):" username_s
  [[ -z "$username_s" ]] && echo "已取消..." && exit 0
  echo && echo "	================================================"
  echo -e "	账号[username]: ${Red_background_prefix} ${username_s} ${Font_color_suffix}"
  echo "	================================================" && echo
}

Set_password() {
  mode=$1
  [[ -z ${mode} ]] && mode="server"
  if [[ ${mode} == "server" ]]; then
    echo -e "请输入 $NAME 服务端要设置的密码[password]（字母/数字，可重复）"
  else
    echo -e "请输入 $NAME 服务端中对应配置的密码[password]（字母/数字）"
  fi
  read -erp "(默认: serverstatus):" password_s
  [[ -z "$password_s" ]] && password_s="serverstatus"
  echo && echo "	================================================"
  echo -e "	密码[password]: ${Red_background_prefix} ${password_s} ${Font_color_suffix}"
  echo "	================================================" && echo
}

Set_name() {
  echo -e "请输入 $NAME 服务端要设置的节点名称[name]（支持中文，前提是你的系统和SSH工具支持中文输入，仅仅是个名字）"
  read -erp "(默认: Server 01):" name_s
  [[ -z "$name_s" ]] && name_s="Server 01"
  echo && echo "	================================================"
  echo -e "	节点名称[name]: ${Red_background_prefix} ${name_s} ${Font_color_suffix}"
  echo "	================================================" && echo
}

Set_type() {
  echo -e "请输入 $NAME 服务端要设置的节点虚拟化类型[type]（例如 OpenVZ / KVM）"
  read -erp "(默认: KVM):" type_s
  [[ -z "$type_s" ]] && type_s="KVM"
  echo && echo "	================================================"
  echo -e "	虚拟化类型[type]: ${Red_background_prefix} ${type_s} ${Font_color_suffix}"
  echo "	================================================" && echo
}

Set_location() {
  echo -e "请输入 $NAME 服务端要设置的节点位置[location]（支持中文，前提是你的系统和SSH工具支持中文输入）"
  read -erp "(默认: Hong Kong):" location_s
  [[ -z "$location_s" ]] && location_s="Hong Kong"
  echo && echo "	================================================"
  echo -e "	节点位置[location]: ${Red_background_prefix} ${location_s} ${Font_color_suffix}"
  echo "	================================================" && echo
}

Set_monthstart() {
  echo -e "请输入 $NAME 服务端要设置的节点月重置流量日[monthstart]（每月流量归零的日期（1~28），默认为1（即每月1日））"
  read -erp "(默认: 1):" monthstart_s
  [[ -z "$monthstart_s" ]] && monthstart_s="1"
  echo && echo "	================================================"
  echo -e "	月流量重置日[monthstart]: ${Red_background_prefix} ${monthstart_s} ${Font_color_suffix}"
  echo "	================================================" && echo
}

Set_config_server() {
  Set_username "server"
  Set_password "server"
  Set_name
  Set_type
  Set_location
  Set_monthstart
}

Set_config_client() {
  Set_server "client"
  Set_server_port
  Set_username "client"
  Set_password "client"
}

Set_ServerStatus_server() {
  check_installed_server_status
  echo && echo -e " 你要做什么？

 ${Green_font_prefix} 1.${Font_color_suffix} 添加 节点配置
 ${Green_font_prefix} 2.${Font_color_suffix} 删除 节点配置
————————
 ${Green_font_prefix} 3.${Font_color_suffix} 修改 节点配置 - 节点用户名
 ${Green_font_prefix} 4.${Font_color_suffix} 修改 节点配置 - 节点密码
 ${Green_font_prefix} 5.${Font_color_suffix} 修改 节点配置 - 节点名称
 ${Green_font_prefix} 6.${Font_color_suffix} 修改 节点配置 - 节点虚拟化
 ${Green_font_prefix} 7.${Font_color_suffix} 修改 节点配置 - 节点位置
 ${Green_font_prefix} 8.${Font_color_suffix} 修改 节点配置 - 月流量重置日
 ${Green_font_prefix} 9.${Font_color_suffix} 修改 节点配置 - 全部参数
————————
 ${Green_font_prefix}10.${Font_color_suffix} 修改 服务端监听端口" && echo
  read -erp "(默认: 取消):" server_num
  [[ -z "${server_num}" ]] && echo "已取消..." && exit 1
  if [[ ${server_num} == "1" ]]; then
    Add_ServerStatus_server
  elif [[ ${server_num} == "2" ]]; then
    Del_ServerStatus_server
  elif [[ ${server_num} == "3" ]]; then
    Modify_ServerStatus_server_username
  elif [[ ${server_num} == "4" ]]; then
    Modify_ServerStatus_server_password
  elif [[ ${server_num} == "5" ]]; then
    Modify_ServerStatus_server_name
  elif [[ ${server_num} == "6" ]]; then
    Modify_ServerStatus_server_type
  elif [[ ${server_num} == "7" ]]; then
    Modify_ServerStatus_server_location
  elif [[ ${server_num} == "8" ]]; then
    Modify_ServerStatus_server_monthstart  
  elif [[ ${server_num} == "9" ]]; then
    Modify_ServerStatus_server_all
  elif [[ ${server_num} == "10" ]]; then
    Read_config_server
    Set_server_port
    Write_server_config_conf
  else
    echo -e "${Error} 请输入正确的数字[1-10]" && exit 1
  fi
  Restart_ServerStatus_server
}

List_ServerStatus_server() {
  conf_text=$(${jq_file} '.servers' ${server_conf} | ${jq_file} ".[]|.username" | sed 's/\"//g')
  conf_text_total=$(echo -e "${conf_text}" | wc -l)
  [[ ${conf_text_total} == "0" ]] && echo -e "${Error} 没有发现 一个节点配置，请检查 !" && exit 1
  conf_text_total_a=$((conf_text_total - 1))
  conf_list_all=""
  for ((integer = 0; integer <= conf_text_total_a; integer++)); do
    now_text=$(${jq_file} '.servers' ${server_conf} | ${jq_file} ".[${integer}]" | sed 's/\"//g;s/,$//g' | sed '$d;1d')
    now_text_username=$(echo -e "${now_text}" | grep "username" | awk -F ": " '{print $2}')
    now_text_password=$(echo -e "${now_text}" | grep "password" | awk -F ": " '{print $2}')
    now_text_name=$(echo -e "${now_text}" | grep "name" | grep -v "username" | awk -F ": " '{print $2}')
    now_text_type=$(echo -e "${now_text}" | grep "type" | awk -F ": " '{print $2}')
    now_text_location=$(echo -e "${now_text}" | grep "location" | awk -F ": " '{print $2}')
    now_text_monthstart=$(echo -e "${now_text}" | grep "monthstart" | awk -F ": " '{print $2}')
    if [[ ${now_text_disabled} == "false" ]]; then
      now_text_disabled_status="${Green_font_prefix}启用${Font_color_suffix}"
    else
      now_text_disabled_status="${Red_font_prefix}禁用${Font_color_suffix}"
    fi
    conf_list_all=${conf_list_all}"用户名: ${Green_font_prefix}${now_text_username}${Font_color_suffix} 密码: ${Green_font_prefix}${now_text_password}${Font_color_suffix} 节点名: ${Green_font_prefix}${now_text_name}${Font_color_suffix} 虚拟化: ${Green_font_prefix}${now_text_type}${Font_color_suffix} 位置: ${Green_font_prefix}${now_text_location}${Font_color_suffix} 月流量重置日: ${Green_font_prefix}${now_text_monthstart}${Font_color_suffix}\n"
  done
  echo && echo -e "节点总数 ${Green_font_prefix}${conf_text_total}${Font_color_suffix}"
  echo -e "${conf_list_all}"
}

Add_ServerStatus_server() {
  Set_config_server
  Set_username_ch=$(grep '"username": "'"${username_s}"'"' ${server_conf})
  [[ -n "${Set_username_ch}" ]] && echo -e "${Error} 用户名已被使用 !" && exit 1
  sed -i '3i\        },' ${server_conf}
  sed -i '3i\            "monthstart": '"${monthstart_s}"'' ${server_conf}
  sed -i '3i\            "location": "'"${location_s}"'",' ${server_conf}
  sed -i '3i\            "host": "'"None"'",' ${server_conf}
  sed -i '3i\            "type": "'"${type_s}"'",' ${server_conf}
  sed -i '3i\            "name": "'"${name_s}"'",' ${server_conf}
  sed -i '3i\            "password": "'"${password_s}"'",' ${server_conf}
  sed -i '3i\            "username": "'"${username_s}"'",' ${server_conf}
  sed -i '3i\        {' ${server_conf}
  echo -e "${Info} 添加节点成功 ${Green_font_prefix}[ 节点名称: ${name_s}, 节点用户名: ${username_s}, 节点密码: ${password_s} ]${Font_color_suffix} !"
}

Del_ServerStatus_server() {
  List_ServerStatus_server
  [[ "${conf_text_total}" == "1" ]] && echo -e "${Error} 节点配置仅剩 1个，不能删除 !" && exit 1
  echo -e "请输入要删除的节点用户名"
  read -erp "(默认: 取消):" del_server_username
  [[ -z "${del_server_username}" ]] && echo -e "已取消..." && exit 1
  del_username=$(cat -n ${server_conf} | grep '"username": "'"${del_server_username}"'"' | awk '{print $1}')
  if [[ -n ${del_username} ]]; then
    del_username_min=$((del_username - 1))
    del_username_max=$((del_username + 7))
    del_username_max_text=$(sed -n "${del_username_max}p" ${server_conf})
    del_username_max_text_last=${del_username_max_text:((${#del_username_max_text} - 1))}
    if [[ ${del_username_max_text_last} != "," ]]; then
      del_list_num=$((del_username_min - 1))
      sed -i "${del_list_num}s/,$//g" ${server_conf}
    fi
    sed -i "${del_username_min},${del_username_max}d" ${server_conf}
    echo -e "${Info} 节点删除成功 ${Green_font_prefix}[ 节点用户名: ${del_server_username} ]${Font_color_suffix} "
  else
    echo -e "${Error} 请输入正确的节点用户名 !" && exit 1
  fi
}

Modify_ServerStatus_server_username() {
  List_ServerStatus_server
  echo -e "请输入要修改的节点用户名"
  read -erp "(默认: 取消):" manually_username
  [[ -z "${manually_username}" ]] && echo -e "已取消..." && exit 1
  Set_username_num=$(cat -n ${server_conf} | grep '"username": "'"${manually_username}"'"' | awk '{print $1}')
  if [[ -n ${Set_username_num} ]]; then
    Set_username
    Set_username_ch=$(grep '"username": "'"${username_s}"'"' ${server_conf})
    [[ -n "${Set_username_ch}" ]] && echo -e "${Error} 用户名已被使用 !" && exit 1
    sed -i "${Set_username_num}"'s/"username": "'"${manually_username}"'"/"username": "'"${username_s}"'"/g' ${server_conf}
    echo -e "${Info} 修改成功 [ 原节点用户名: ${manually_username}, 新节点用户名: ${username_s} ]"
  else
    echo -e "${Error} 请输入正确的节点用户名 !" && exit 1
  fi
}

Modify_ServerStatus_server_password() {
  List_ServerStatus_server
  echo -e "请输入要修改的节点用户名"
  read -erp "(默认: 取消):" manually_username
  [[ -z "${manually_username}" ]] && echo -e "已取消..." && exit 1
  Set_username_num=$(cat -n ${server_conf} | grep '"username": "'"${manually_username}"'"' | awk '{print $1}')
  if [[ -n ${Set_username_num} ]]; then
    Set_password
    Set_password_num_a=$((Set_username_num + 1))
    Set_password_num_text=$(sed -n "${Set_password_num_a}p" ${server_conf} | sed 's/\"//g;s/,$//g' | awk -F ": " '{print $2}')
    sed -i "${Set_password_num_a}"'s/"password": "'"${Set_password_num_text}"'"/"password": "'"${password_s}"'"/g' ${server_conf}
    echo -e "${Info} 修改成功 [ 原节点密码: ${Set_password_num_text}, 新节点密码: ${password_s} ]"
  else
    echo -e "${Error} 请输入正确的节点用户名 !" && exit 1
  fi
}

Modify_ServerStatus_server_name() {
  List_ServerStatus_server
  echo -e "请输入要修改的节点用户名"
  read -erp "(默认: 取消):" manually_username
  [[ -z "${manually_username}" ]] && echo -e "已取消..." && exit 1
  Set_username_num=$(cat -n ${server_conf} | grep '"username": "'"${manually_username}"'"' | awk '{print $1}')
  if [[ -n ${Set_username_num} ]]; then
    Set_name
    Set_name_num_a=$((Set_username_num + 2))
    Set_name_num_a_text=$(sed -n "${Set_name_num_a}p" ${server_conf} | sed 's/\"//g;s/,$//g' | awk -F ": " '{print $2}')
    sed -i "${Set_name_num_a}"'s/"name": "'"${Set_name_num_a_text}"'"/"name": "'"${name_s}"'"/g' ${server_conf}
    echo -e "${Info} 修改成功 [ 原节点名称: ${Set_name_num_a_text}, 新节点名称: ${name_s} ]"
  else
    echo -e "${Error} 请输入正确的节点用户名 !" && exit 1
  fi
}

Modify_ServerStatus_server_type() {
  List_ServerStatus_server
  echo -e "请输入要修改的节点用户名"
  read -erp "(默认: 取消):" manually_username
  [[ -z "${manually_username}" ]] && echo -e "已取消..." && exit 1
  Set_username_num=$(cat -n ${server_conf} | grep '"username": "'"${manually_username}"'"' | awk '{print $1}')
  if [[ -n ${Set_username_num} ]]; then
    Set_type
    Set_type_num_a=$((Set_username_num + 3))
    Set_type_num_a_text=$(sed -n "${Set_type_num_a}p" ${server_conf} | sed 's/\"//g;s/,$//g' | awk -F ": " '{print $2}')
    sed -i "${Set_type_num_a}"'s/"type": "'"${Set_type_num_a_text}"'"/"type": "'"${type_s}"'"/g' ${server_conf}
    echo -e "${Info} 修改成功 [ 原节点虚拟化: ${Set_type_num_a_text}, 新节点虚拟化: ${type_s} ]"
  else
    echo -e "${Error} 请输入正确的节点用户名 !" && exit 1
  fi
}

Modify_ServerStatus_server_location() {
  List_ServerStatus_server
  echo -e "请输入要修改的节点用户名"
  read -erp "(默认: 取消):" manually_username
  [[ -z "${manually_username}" ]] && echo -e "已取消..." && exit 1
  Set_username_num=$(cat -n ${server_conf} | grep '"username": "'"${manually_username}"'"' | awk '{print $1}')
  if [[ -n ${Set_username_num} ]]; then
    Set_location
    Set_location_num_a=$((Set_username_num + 5))
    Set_location_num_a_text=$(sed -n "${Set_location_num_a}p" ${server_conf} | sed 's/\"//g;s/,$//g' | awk -F ": " '{print $2}')
    sed -i "${Set_location_num_a}"'s/"location": "'"${Set_location_num_a_text}"'"/"location": "'"${location_s}"'"/g' ${server_conf}
    echo -e "${Info} 修改成功 [ 原节点位置: ${Set_location_num_a_text}, 新节点位置: ${location_s} ]"
  else
    echo -e "${Error} 请输入正确的节点用户名 !" && exit 1
  fi
}

Modify_ServerStatus_server_monthstart() {
  List_ServerStatus_server
  echo -e "请输入要修改的节点用户名"
  read -erp "(默认: 取消):" manually_username
  [[ -z "${manually_username}" ]] && echo -e "已取消..." && exit 1
  Set_username_num=$(cat -n ${server_conf} | grep '"username": "'"${manually_username}"'"' | awk '{print $1}')
  if [[ -n ${Set_username_num} ]]; then
    Set_monthstart
    Set_monthstart_num_a=$((Set_username_num + 6))
    Set_monthstart_num_text=$(sed -n "${Set_monthstart_num_a}p" ${server_conf} | sed 's/\"//g;s/,$//g' | awk -F ": " '{print $2}')
    sed -i "${Set_monthstart_num_a}"'s/"monthstart": '"${Set_monthstart_num_text}"'/"monthstart": '"${monthstart_s}"'/g' ${server_conf}
    echo -e "${Info} 修改成功 [ 原月流量重置日: ${Set_monthstart_num_text}, 新月流量重置日: ${monthstart_s} ]"
  else
    echo -e "${Error} 请输入正确的节点用户名 !" && exit 1
  fi
}

Modify_ServerStatus_server_all() {
  List_ServerStatus_server
  echo -e "请输入要修改的节点用户名"
  read -erp "(默认: 取消):" manually_username
  [[ -z "${manually_username}" ]] && echo -e "已取消..." && exit 1
  Set_username_num=$(cat -n ${server_conf} | grep '"username": "'"${manually_username}"'"' | awk '{print $1}')
  if [[ -n ${Set_username_num} ]]; then
    Set_username
    Set_password
    Set_name
    Set_type
    Set_location
    Set_monthstart
    sed -i "${Set_username_num}"'s/"username": "'"${manually_username}"'"/"username": "'"${username_s}"'"/g' ${server_conf}
    Set_password_num_a=$((Set_username_num + 1))
    Set_password_num_text=$(sed -n "${Set_password_num_a}p" ${server_conf} | sed 's/\"//g;s/,$//g' | awk -F ": " '{print $2}')
    sed -i "${Set_password_num_a}"'s/"password": "'"${Set_password_num_text}"'"/"password": "'"${password_s}"'"/g' ${server_conf}
    Set_name_num_a=$((Set_username_num + 2))
    Set_name_num_a_text=$(sed -n "${Set_name_num_a}p" ${server_conf} | sed 's/\"//g;s/,$//g' | awk -F ": " '{print $2}')
    sed -i "${Set_name_num_a}"'s/"name": "'"${Set_name_num_a_text}"'"/"name": "'"${name_s}"'"/g' ${server_conf}
    Set_type_num_a=$((Set_username_num + 3))
    Set_type_num_a_text=$(sed -n "${Set_type_num_a}p" ${server_conf} | sed 's/\"//g;s/,$//g' | awk -F ": " '{print $2}')
    sed -i "${Set_type_num_a}"'s/"type": "'"${Set_type_num_a_text}"'"/"type": "'"${type_s}"'"/g' ${server_conf}
    Set_location_num_a=$((Set_username_num + 5))
    Set_location_num_a_text=$(sed -n "${Set_location_num_a}p" ${server_conf} | sed 's/\"//g;s/,$//g' | awk -F ": " '{print $2}')
    sed -i "${Set_location_num_a}"'s/"location": "'"${Set_location_num_a_text}"'"/"location": "'"${location_s}"'"/g' ${server_conf}
    Set_monthstart_num_a=$((Set_username_num + 6))
    Set_monthstart_num_a_text=$(sed -n "${Set_monthstart_num_a}p" ${server_conf} | sed 's/\"//g;s/,$//g' | awk -F ": " '{print $2}')
    sed -i "${Set_monthstart_num_a}"'s/"monthstart": '"${Set_monthstart_num_a_text}"'/"monthstart": '"${monthstart_s}"'/g' ${server_conf}
    echo -e "${Info} 修改成功。"
  else
    echo -e "${Error} 请输入正确的节点用户名 !" && exit 1
  fi
}

Modify_ServerStatus_server_disabled() {
  List_ServerStatus_server
  echo -e "请输入要修改的节点用户名"
  read -erp "(默认: 取消):" manually_username
  [[ -z "${manually_username}" ]] && echo -e "已取消..." && exit 1
  Set_username_num=$(cat -n ${server_conf} | grep '"username": "'"${manually_username}"'"' | awk '{print $1}')
  if [[ -n ${Set_username_num} ]]; then
    Set_disabled_num_a=$((Set_username_num + 6))
    Set_disabled_num_a_text=$(sed -n "${Set_disabled_num_a}p" ${server_conf} | sed 's/\"//g;s/,$//g' | awk -F ": " '{print $2}')
    if [[ ${Set_disabled_num_a_text} == "false" ]]; then
      disabled_s="true"
    else
      disabled_s="false"
    fi
    sed -i "${Set_disabled_num_a}"'s/"disabled": '"${Set_disabled_num_a_text}"'/"disabled": '"${disabled_s}"'/g' ${server_conf}
    echo -e "${Info} 修改成功 [ 原禁用状态: ${Set_disabled_num_a_text}, 新禁用状态: ${disabled_s} ]"
  else
    echo -e "${Error} 请输入正确的节点用户名 !" && exit 1
  fi
}

Set_ServerStatus_client() {
  check_installed_client_status
  Set_config_client
  Read_config_client
  Modify_config_client
  Restart_ServerStatus_client
}

Modify_config_client() {
  sed -i '0,/SERVER = "'"${client_server}"'"/s//SERVER = "'"${server_s}"'"/' "${client_file}/client-linux.py"
  sed -i '0,/PORT = ${client_port}/s//PORT = ${server_port_s}/' "${client_file}/client-linux.py"
  sed -i '0,/USER = "'"${client_user}"'"/s//USER = "'"${username_s}"'"/' "${client_file}/client-linux.py"
  sed -i '0,/PASSWORD = "'"${client_password}"'"/s//PASSWORD = "'"${password_s}"'"/' "${client_file}/client-linux.py"
}

Install_jq() {
  [[ ${mirror_num} == 2 ]] && {
    github_link="https://hub.fastgit.org"
    raw_link="https://raw.fastgit.org"
  } || {
    github_link="https://github.com"
    raw_link="https://raw.githubusercontent.com"
  }
  if [[ ! -e ${jq_file} ]]; then
    if [[ ${bit} == "x86_64" ]]; then
      jq_file="${file}/jq"
      wget --no-check-certificate "${github_link}/stedolan/jq/releases/download/jq-1.5/jq-linux64" -O ${jq_file}
    elif [[ ${bit} == "i386" ]]; then
      jq_file="${file}/jq"
      wget --no-check-certificate "${github_link}/stedolan/jq/releases/download/jq-1.5/jq-linux32" -O ${jq_file}
    else
      # ARM fallback to package manager
      [[ ${release} == "archlinux" ]] && pacman -Sy jq --noconfirm
      [[ ${release} == "centos" ]] && yum -y install jq
      [[ ${release} == "debian" ]] && apt -y install jq
      jq_file="/usr/bin/jq"
    fi
    [[ ! -e ${jq_file} ]] && echo -e "${Error} JQ解析器 下载失败，请检查 !" && exit 1
    chmod +x ${jq_file}
    echo -e "${Info} JQ解析器 安装完成，继续..."
  else
    echo -e "${Info} JQ解析器 已安装，继续..."
  fi
}

Install_caddy() {
  echo
  echo -e "${Info} 是否由脚本自动配置HTTP服务(服务端的在线监控网站)，如果选择 N，则请在其他HTTP服务中配置网站根目录为：${Green_font_prefix}${web_file}${Font_color_suffix} [Y/n]"
  read -erp "(默认: Y 自动部署):" caddy_yn
  [[ -z "$caddy_yn" ]] && caddy_yn="y"
  if [[ "${caddy_yn}" == [Yy] ]]; then
    caddy_file="/etc/caddy/Caddyfile" # Where is the default Caddyfile specified in Archlinux?
    [[ ! -e /usr/bin/caddy ]] && {
      if [[ ${release} == "debian" ]]; then
        apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
        curl -1sLf "https://dl.cloudsmith.io/public/caddy/stable/gpg.key" | tee /etc/apt/trusted.gpg.d/caddy-stable.asc
        curl -1sLf "https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt" | tee /etc/apt/sources.list.d/caddy-stable.list
        apt update && apt install caddy
      elif [[ ${release} == "centos" ]]; then
        yum install yum-plugin-copr -y
        yum copr enable @caddy/caddy -y
        yum install caddy -y
      elif [[ ${release} == "archlinux" ]]; then
        pacman -Sy caddy --noconfirm
      fi
      [[ ! -e "/usr/bin/caddy" ]] && echo -e "${Error} Caddy安装失败，请手动部署，Web网页文件位置：${web_file}" && exit 1
      systemctl enable caddy
      echo "" >${caddy_file}
    }
    Set_server "server"
    Set_server_http_port
    cat >>${caddy_file} <<-EOF
http://${server_s}:${server_http_port_s} {
  root * ${web_file}
  encode gzip
  file_server
}
EOF
    systemctl restart caddy
  else
    echo -e "${Info} 跳过 HTTP服务部署，请手动部署，Web网页文件位置：${web_file} ，如果位置改变，请注意修改服务脚本文件 /etc/init.d/status-server 中的 WEB_BIN 变量 !"
  fi
}

Install_ServerStatus_server() {
  [[ -e "${server_file}/sergate" ]] && echo -e "${Error} 检测到 $NAME 服务端已安装 !" && exit 1
  Set_server_port
  echo -e "${Info} 开始安装/配置 依赖..."
  Installation_dependency "server"
  Install_caddy
  echo -e "${Info} 开始下载/安装..."
  Download_Server_Status_server
Install_jq
  echo -e "${Info} 开始下载/安装 服务脚本..."
  Service_Server_Status_server
  echo -e "${Info} 开始写入 配置文件..."
  Write_server_config
  Write_server_config_conf
  echo -e "${Info} 所有步骤 安装完毕，开始启动..."
  Start_ServerStatus_server
}

Install_ServerStatus_client() {
  [[ -e "${client_file}/client-linux.py" ]] && echo -e "${Error} 检测到 $NAME 客户端已安装 !" && exit 1
  check_sys
  echo -e "${Info} 开始设置 用户配置..."
  Set_config_client
  echo -e "${Info} 开始安装/配置 依赖..."
  Installation_dependency "client"
  echo -e "${Info} 开始下载/安装..."
  Download_Server_Status_client
  echo -e "${Info} 开始下载/安装 服务脚本..."
  Service_Server_Status_client
  echo -e "${Info} 开始写入 配置..."
  Read_config_client
  Modify_config_client
  echo -e "${Info} 所有步骤 安装完毕，开始启动..."
  Start_ServerStatus_client
}

Update_ServerStatus_server() {
  check_installed_server_status
  Download_Server_Status_server
  rm -rf /etc/init.d/status-server
  Service_Server_Status_server
  Start_ServerStatus_server
}

Update_ServerStatus_client() {
  check_installed_client_status
  systemctl stop status-client
  client_text="$(sed 's/\"//g;s/,//g;s/ //g' "${client_file}/client-linux.py")"
  server_s="$(echo -e "${client_text}" | grep "SERVER=" | awk -F "=" '{print $2;exit}')"
  server_port_s="$(echo -e "${client_text}" | grep "PORT=" | awk -F "=" '{print $2;exit}')"
  username_s="$(echo -e "${client_text}" | grep "USER=" | awk -F "=" '{print $2;exit}')"
  password_s="$(echo -e "${client_text}" | grep "PASSWORD=" | awk -F "=" '{print $2;exit}')"
  Download_Server_Status_client
  Read_config_client
  Modify_config_client
  rm -rf  ${service}/status-client.service
  Service_Server_Status_client
  Start_ServerStatus_client
}

Start_ServerStatus_server() {
  port="$(grep "m_Port = " ${server_file}/src/main.cpp | awk '{print $3}' | sed '{s/;$//}')"
  check_installed_server_status
  systemctl -q is-active status-server && echo -e "${Error} $NAME 正在运行，请检查 !" && exit 1
  systemctl start status-server
		if (systemctl -q is-active status-server) then
			echo -e "${Info} $NAME 服务端启动成功[监听端口：${port}] !"
		else
			echo -e "${Error} $NAME 服务端启动失败 !"
		fi
}

Stop_ServerStatus_server() {
  check_installed_server_status
if (systemctl -q is-active status-server)
  then
  systemctl stop status-server 
 else  
 echo -e "${Error} $NAME 没有运行，请检查 !" && exit 1
fi
		if (systemctl -q is-active status-server) then
			echo -e "${Error} $NAME 服务端停止失败 !"
		else
			echo -e "${Info} $NAME 服务端停止成功 !"
		fi
}

Restart_ServerStatus_server() {
  check_installed_server_status
  systemctl restart status-server
if (systemctl -q is-active status-server)
     then
     echo -e "${Info} $NAME 服务端重启成功 !"
else
     echo -e "${Error} $NAME 服务端重启失败 !" && exit 1
fi
}

Uninstall_ServerStatus_server() {
  check_installed_server_status
  echo "确定要卸载 $NAME 服务端(如果同时安装了客户端，则只会删除服务端) ? [y/N]"
  echo
  read -erp "(默认: n):" unyn
  [[ -z ${unyn} ]] && unyn="n"
  if [[ ${unyn} == [Yy] ]]; then
    systemctl stop status-server
    systemctl disable status-server
    rm ${service}/status-server.service -f
    if [[ -e "${client_file}/client-linux.py" ]]; then
      rm -rf "${server_file}"
      rm -rf "${web_file}"
      rm -rf "${plugin_file}"
    else
      rm -rf "${file}"
    fi
    if [[ -e "/usr/bin/caddy" ]]; then
      systemctl stop caddy
      systemctl disable caddy
      [[ ${release} == "debian" ]] && apt purge -y caddy
      [[ ${release} == "centos" ]] && yum -y remove caddy
      [[ ${release} == "archlinux" ]] && pacman -R caddy --noconfirm
    fi
    systemctl daemon-reload
    systemctl reset-failed
    echo && echo "ServerStatus 卸载完成 !" && echo
  else
    echo && echo "卸载已取消..." && echo
  fi
}

Start_ServerStatus_client() {
  check_installed_client_status
if (systemctl -q is-active status-client) then
    echo -e "${Error} $NAME 客户端正在运行，请检查 !" && exit 1
fi
   systemctl start status-client
   if (systemctl -q is-active status-client)
     then
       echo -e "${Info} $NAME 客户端启动成功 !"
   else
       echo -e "${Error} $NAME 客户端启动失败 !"
   fi
}

Stop_ServerStatus_client() {
  check_installed_client_status
if (systemctl -q is-active status-client) then
  systemctl stop status-client
    if (systemctl -q is-active status-client) then
       echo -e "${Error}} $NAME 停止失败 !"
      else
       echo -e "${Info} $NAME 停止成功 !"
    fi
else
    echo -e "${Error} $NAME 没有运行，请检查 !" && exit 1
fi
}

Restart_ServerStatus_client() {
  systemctl restart status-client
if (systemctl -q is-active status-client) then
     echo -e "${Info} $NAME 重启成功 !"
else
     echo -e "${Error} $NAME 重启失败 !" && exit 1
fi
}

Uninstall_ServerStatus_client() {
  check_installed_client_status
  echo "确定要卸载 $NAME 客户端(如果同时安装了服务端，则只会删除客户端) ? [y/N]"
  echo
  read -erp "(默认: n):" unyn
  [[ -z ${unyn} ]] && unyn="n"
  if [[ ${unyn} == [Yy] ]]; then
    systemctl stop status-client
    systemctl disable status-client
    rm ${service}/status-client.service -f
    systemctl daemon-reload
    systemctl reset-failed
    rm -rf "${client_file}"
    echo && echo "ServerStatus 卸载完成 !" && echo
  else
    echo && echo "卸载已取消..." && echo
  fi
}

View_ServerStatus_client() {
  check_installed_client_status
  Read_config_client
  clear && echo "————————————————————" && echo
  echo -e "  $NAME 客户端配置信息：

  IP \t: ${Green_font_prefix}${client_server}${Font_color_suffix}
  端口 \t: ${Green_font_prefix}${client_port}${Font_color_suffix}
  账号 \t: ${Green_font_prefix}${client_user}${Font_color_suffix}
  密码 \t: ${Green_font_prefix}${client_password}${Font_color_suffix}

————————————————————"
}

View_client_Log() {
    journalctl -u status-client.service --no-pager -f
    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

View_server_Log() {
    journalctl -u status-server.service --no-pager -f
    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

Update_Shell() {
  sh_new_ver=$(wget --no-check-certificate -qO- -t1 -T3 "${github_prefix}/status.sh" | grep 'sh_ver="' | awk -F "=" '{print $NF}' | sed 's/\"//g' | head -1)
  [[ -z ${sh_new_ver} ]] && echo -e "${Error} 无法链接到 Github !" && exit 0
  if  [[ -e "${service}/status-client.service" ]]; then
    rm -rf ${service}/status-client.service
    Service_Server_Status_client
  fi
  if  [[ -e "${service}/status-server.service" ]]; then
    rm -rf ${service}/status-server.service
    Service_Server_Status_server
  fi
  wget -N --no-check-certificate "${github_prefix}/status.sh"
  echo -e "脚本已更新为最新版本[ ${sh_new_ver} ] !(注意：因为更新方式为直接覆盖当前运行的脚本，所以可能下面会提示一些报错，无视即可)" && exit 0
}

menu_client() {
  echo && echo -e "  $NAME 一键安装管理脚本 ${Red_font_prefix}[v${sh_ver}]${Font_color_suffix}

 ${Green_font_prefix} 0.${Font_color_suffix} 升级脚本
 ————————————
 ${Green_font_prefix} 1.${Font_color_suffix} 安装 客户端
 ${Green_font_prefix} 2.${Font_color_suffix} 更新 客户端
 ${Green_font_prefix} 3.${Font_color_suffix} 卸载 客户端
————————————
 ${Green_font_prefix} 4.${Font_color_suffix} 启动 客户端
 ${Green_font_prefix} 5.${Font_color_suffix} 停止 客户端
 ${Green_font_prefix} 6.${Font_color_suffix} 重启 客户端
————————————
 ${Green_font_prefix} 7.${Font_color_suffix} 设置 客户端配置
 ${Green_font_prefix} 8.${Font_color_suffix} 查看 客户端信息
 ${Green_font_prefix} 9.${Font_color_suffix} 查看 客户端日志
————————————
 ${Green_font_prefix}10.${Font_color_suffix} 切换为 服务端菜单" && echo
  if [[ -e "${client_file}/client-linux.py" ]]; then
    if (systemctl -q is-active status-client); then
      echo -e " 当前状态: 客户端 ${Green_font_prefix}已安装${Font_color_suffix} 并 ${Green_font_prefix}已启动${Font_color_suffix}"
    else
      echo -e " 当前状态: 客户端 ${Green_font_prefix}已安装${Font_color_suffix} 但 ${Red_font_prefix}未启动${Font_color_suffix}"
    fi
    else
      echo -e " 当前状态: 客户端 ${Red_font_prefix}未安装${Font_color_suffix}"
  fi
  echo
  read -erp " 请输入数字 [0-10]:" num
  case "$num" in
  0)
    Update_Shell
    ;;
  1)
    Install_ServerStatus_client
    ;;
  2)
    Update_ServerStatus_client
    ;;
  3)
    Uninstall_ServerStatus_client
    ;;
  4)
    Start_ServerStatus_client
    ;;
  5)
    Stop_ServerStatus_client
    ;;
  6)
    Restart_ServerStatus_client
    ;;
  7)
    Set_ServerStatus_client
    ;;
  8)
    View_ServerStatus_client
    ;;
  9)
    View_client_Log
    ;;
  10)
    menu_server
    ;;
  *)
    echo "请输入正确数字 [0-10]"
    ;;
  esac
}
menu_server() {
  echo && echo -e "  $NAME 一键安装管理脚本 ${Red_font_prefix}[v${sh_ver}]${Font_color_suffix}

 ${Green_font_prefix} 0.${Font_color_suffix} 升级脚本
 ————————————
 ${Green_font_prefix} 1.${Font_color_suffix} 安装 服务端
 ${Green_font_prefix} 2.${Font_color_suffix} 更新 服务端
 ${Green_font_prefix} 3.${Font_color_suffix} 卸载 服务端
————————————
 ${Green_font_prefix} 4.${Font_color_suffix} 启动 服务端
 ${Green_font_prefix} 5.${Font_color_suffix} 停止 服务端
 ${Green_font_prefix} 6.${Font_color_suffix} 重启 服务端
————————————
 ${Green_font_prefix} 7.${Font_color_suffix} 设置 服务端配置
 ${Green_font_prefix} 8.${Font_color_suffix} 查看 服务端信息
 ${Green_font_prefix} 9.${Font_color_suffix} 查看 服务端日志
————————————
 ${Green_font_prefix}10.${Font_color_suffix} 切换为 客户端菜单" && echo
  if [[ -e "${server_file}/sergate" ]]; then
    if (systemctl -q is-active status-server) then
      echo -e " 当前状态: 服务端 ${Green_font_prefix}已安装${Font_color_suffix} 并 ${Green_font_prefix}已启动${Font_color_suffix}"
    else
      echo -e " 当前状态: 服务端 ${Green_font_prefix}已安装${Font_color_suffix} 但 ${Red_font_prefix}未启动${Font_color_suffix}"
    fi
  else
    echo -e " 当前状态: 服务端 ${Red_font_prefix}未安装${Font_color_suffix}"
  fi
  echo
  read -erp " 请输入数字 [0-10]:" num
  case "$num" in
  0)
    Update_Shell
    ;;
  1)
    Install_ServerStatus_server
    ;;
  2)
    Update_ServerStatus_server
    ;;
  3)
    Uninstall_ServerStatus_server
    ;;
  4)
    Start_ServerStatus_server
    ;;
  5)
    Stop_ServerStatus_server
    ;;
  6)
    Restart_ServerStatus_server
    ;;
  7)
    Set_ServerStatus_server
    ;;
  8)
    List_ServerStatus_server
    ;;
  9)
    View_server_Log
    ;;
  10)
    menu_client
    ;;
  *)
    echo "请输入正确数字 [0-10]"
    ;;
  esac
}

check_sys
action=$1
if [[ -n $action ]]; then
  if [[ $action == "s" ]]; then
    menu_server
  elif [[ $action == "c" ]]; then
    menu_client
  fi
else
  menu_client
fi
