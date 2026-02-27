#!/usr/bin/env python3
# coding: utf-8
# Update by : https://github.com/cppla/ServerStatus, Update date: 20250902
# 版本：1.1.0, 支持Python版本：3.6+
# 支持操作系统： Linux, OSX, FreeBSD, OpenBSD and NetBSD, both 32-bit and 64-bit architectures
# 说明: 默认情况下修改server和user就可以了。丢包率监测方向可以自定义，例如：CU = "www.facebook.com"。

SERVER = "127.0.0.1"
USER = "s01"


PASSWORD = "USER_DEFAULT_PASSWORD"
PORT = 35601
CU = "cu.tz.cloudcpp.com"
CT = "ct.tz.cloudcpp.com"
CM = "cm.tz.cloudcpp.com"
PROBEPORT = 80
PROBE_PROTOCOL_PREFER = "ipv4"  # ipv4, ipv6
PING_PACKET_HISTORY_LEN = 100
INTERVAL = 1

import socket
import time
import timeit
import re
import os
import sys
import json
import errno
import subprocess
import threading
import platform
from queue import Queue

def _env_str(name, default):
    value = os.getenv(name)
    if value is None or value == "":
        return default
    return value

def _env_int(name, default):
    value = os.getenv(name)
    if value is None or value == "":
        return default
    try:
        return int(value)
    except ValueError:
        return default

# Allow docker env overrides
SERVER = _env_str("SERVER", SERVER)
USER = _env_str("USER", USER)
PASSWORD = _env_str("PASSWORD", PASSWORD)
PORT = _env_int("PORT", PORT)
INTERVAL = _env_int("INTERVAL", INTERVAL)
PROBEPORT = _env_int("PROBEPORT", PROBEPORT)
PROBE_PROTOCOL_PREFER = _env_str("PROBE_PROTOCOL_PREFER", PROBE_PROTOCOL_PREFER)
PING_PACKET_HISTORY_LEN = _env_int("PING_PACKET_HISTORY_LEN", PING_PACKET_HISTORY_LEN)
CU = _env_str("CU", CU)
CT = _env_str("CT", CT)
CM = _env_str("CM", CM)

def get_uptime():
    with open('/proc/uptime', 'r') as f:
        uptime = f.readline().split('.', 2)
        return int(uptime[0])

def get_memory():
    re_parser = re.compile(r'^(?P<key>\S*):\s*(?P<value>\d*)\s*kB')
    result = dict()
    for line in open('/proc/meminfo'):
        match = re_parser.match(line)
        if not match:
            continue
        key, value = match.groups(['key', 'value'])
        result[key] = int(value)
    MemTotal = float(result['MemTotal'])
    MemUsed = MemTotal-float(result['MemFree'])-float(result['Buffers'])-float(result['Cached'])-float(result['SReclaimable'])
    SwapTotal = float(result['SwapTotal'])
    SwapFree = float(result['SwapFree'])
    return int(MemTotal), int(MemUsed), int(SwapTotal), int(SwapFree)

def get_hdd():
    valid_fs = {
        "ext4", "ext3", "ext2", "reiserfs", "jfs", "ntfs", "fat32",
        "btrfs", "fuseblk", "zfs", "simfs", "xfs", "exfat", "f2fs"
    }
    seen = set()
    size = 0
    used = 0
    try:
        with open("/proc/mounts", "r") as f:
            for line in f:
                parts = line.split()
                if len(parts) < 3:
                    continue
                mountpoint = parts[1]
                fstype = parts[2].lower()
                if fstype not in valid_fs or mountpoint in seen:
                    continue
                seen.add(mountpoint)
                st = os.statvfs(mountpoint)
                total_bytes = st.f_blocks * st.f_frsize
                used_bytes = (st.f_blocks - st.f_bavail) * st.f_frsize
                size += total_bytes
                used += used_bytes
    except Exception:
        pass
    # Keep client alive even on minimal/containerized systems.
    if size <= 0:
        st = os.statvfs("/")
        size = st.f_blocks * st.f_frsize
        used = (st.f_blocks - st.f_bavail) * st.f_frsize
    return int(size / 1024 / 1024), int(used / 1024 / 1024)

def get_time():
    with open("/proc/stat", "r") as f:
        time_list = f.readline().split(' ')[2:6]
        for i in range(len(time_list))  :
            time_list[i] = int(time_list[i])
        return time_list

def delta_time():
    x = get_time()
    time.sleep(INTERVAL)
    y = get_time()
    for i in range(len(x)):
        y[i]-=x[i]
    return y

def get_cpu():
    t = delta_time()
    st = sum(t)
    if st == 0:
        st = 1
    result = 100-(t[len(t)-1]*100.00/st)
    return round(result, 1)

def liuliang():
    NET_IN = 0
    NET_OUT = 0
    with open('/proc/net/dev') as f:
        for line in f.readlines():
            netinfo = re.findall(r'([^\s]+):[\s]{0,}(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)', line)
            if netinfo:
                if netinfo[0][0] == 'lo' or 'tun' in netinfo[0][0] \
                        or 'docker' in netinfo[0][0] or 'veth' in netinfo[0][0] \
                        or 'br-' in netinfo[0][0] or 'vmbr' in netinfo[0][0] \
                        or 'vnet' in netinfo[0][0] or 'kube' in netinfo[0][0] \
                        or netinfo[0][1]=='0' or netinfo[0][9]=='0':
                    continue
                else:
                    NET_IN += int(netinfo[0][1])
                    NET_OUT += int(netinfo[0][9])
    return NET_IN, NET_OUT

def tupd():
    '''
    tcp, udp, process, thread count: for view ddcc attack , then send warning
    :return:
    '''
    s = subprocess.check_output("ss -t|wc -l", shell=True)
    t = int(s[:-1])-1
    s = subprocess.check_output("ss -u|wc -l", shell=True)
    u = int(s[:-1])-1
    s = subprocess.check_output("ps -ef|wc -l", shell=True)
    p = int(s[:-1])-2
    s = subprocess.check_output("ps -eLf|wc -l", shell=True)
    d = int(s[:-1])-2
    return t,u,p,d

def get_network(ip_version):
    if(ip_version == 4):
        HOST = "ipv4.google.com"
    elif(ip_version == 6):
        HOST = "ipv6.google.com"
    try:
        socket.create_connection((HOST, 80), 2).close()
        return True
    except:
        return False

lostRate = {
    '10010': 0.0,
    '189': 0.0,
    '10086': 0.0
}
pingTime = {
    '10010': 0,
    '189': 0,
    '10086': 0
}
netSpeed = {
    'netrx': 0.0,
    'nettx': 0.0,
    'clock': 0.0,
    'diff': 0.0,
    'avgrx': 0,
    'avgtx': 0
}
diskIO = {
    'read': 0,
    'write': 0
}
monitorServer = {}

def _ping_thread(host, mark, port):
    lostPacket = 0
    packet_queue = Queue(maxsize=PING_PACKET_HISTORY_LEN)

    while True:
        # flush dns , every time.
        IP = host
        if host.count(':') < 1:  # if not plain ipv6 address, means ipv4 address or hostname
            try:
                if PROBE_PROTOCOL_PREFER == 'ipv4':
                    IP = socket.getaddrinfo(host, None, socket.AF_INET)[0][4][0]
                else:
                    IP = socket.getaddrinfo(host, None, socket.AF_INET6)[0][4][0]
            except Exception:
                pass

        if packet_queue.full():
            if packet_queue.get() == 0:
                lostPacket -= 1
        try:
            b = timeit.default_timer()
            socket.create_connection((IP, port), timeout=1).close()
            pingTime[mark] = int((timeit.default_timer() - b) * 1000)
            packet_queue.put(1)
        except socket.error as error:
            if error.errno == errno.ECONNREFUSED:
                pingTime[mark] = int((timeit.default_timer() - b) * 1000)
                packet_queue.put(1)
            #elif error.errno == errno.ETIMEDOUT:
            else:
                lostPacket += 1
                packet_queue.put(0)

        if packet_queue.qsize() > 30:
            lostRate[mark] = float(lostPacket) / packet_queue.qsize()

        time.sleep(INTERVAL)

def _net_speed():
    while True:
        with open("/proc/net/dev", "r") as f:
            net_dev = f.readlines()
            avgrx = 0
            avgtx = 0
            for dev in net_dev[2:]:
                dev = dev.split(':')
                if "lo" in dev[0] or "tun" in dev[0] \
                        or "docker" in dev[0] or "veth" in dev[0] \
                        or "br-" in dev[0] or "vmbr" in dev[0] \
                        or "vnet" in dev[0] or "kube" in dev[0]:
                    continue
                dev = dev[1].split()
                avgrx += int(dev[0])
                avgtx += int(dev[8])
            now_clock = time.time()
            netSpeed["diff"] = now_clock - netSpeed["clock"]
            netSpeed["clock"] = now_clock
            netSpeed["netrx"] = int((avgrx - netSpeed["avgrx"]) / netSpeed["diff"])
            netSpeed["nettx"] = int((avgtx - netSpeed["avgtx"]) / netSpeed["diff"])
            netSpeed["avgrx"] = avgrx
            netSpeed["avgtx"] = avgtx
        time.sleep(INTERVAL)

def _disk_io():
    '''
    good luck for opensource! by: cpp.la
    磁盘IO：因为IOPS原因，SSD和HDD、包括RAID卡，ZFS等阵列技术。IO对性能的影响还需要结合自身服务器情况来判断。
    比如我这里是机械硬盘，大量做随机小文件读写，那么很低的读写也就能造成硬盘长时间的等待。
    如果这里做连续性IO，那么普通机械硬盘写入到100Mb/s，那么也能造成硬盘长时间的等待。
    磁盘读写有误差：4k，8k ，https://stackoverflow.com/questions/34413926/psutil-vs-dd-monitoring-disk-i-o
    :return:
    '''
    while True:
        # pre pid snapshot
        snapshot_first = {}
        # next pid snapshot
        snapshot_second = {}
        # read count snapshot
        snapshot_read = 0
        # write count snapshot
        snapshot_write = 0
        # process snapshot
        pid_snapshot = [str(i) for i in os.listdir("/proc") if i.isdigit() is True]
        for pid in pid_snapshot:
            try:
                with open("/proc/{}/io".format(pid)) as f:
                    pid_io = {}
                    for line in f.readlines():
                        if "read_bytes" in line:
                            pid_io["read"] = int(line.split("read_bytes:")[-1].strip())
                        elif "write_bytes" in line and "cancelled_write_bytes" not in line:
                            pid_io["write"] = int(line.split("write_bytes:")[-1].strip())
                    pid_io["name"] = open("/proc/{}/comm".format(pid), "r").read().strip()
                    snapshot_first[pid] = pid_io
            except:
                if pid in snapshot_first:
                    snapshot_first.pop(pid)

        time.sleep(INTERVAL)

        for pid in pid_snapshot:
            try:
                with open("/proc/{}/io".format(pid)) as f:
                    pid_io = {}
                    for line in f.readlines():
                        if "read_bytes" in line:
                            pid_io["read"] = int(line.split("read_bytes:")[-1].strip())
                        elif "write_bytes" in line and "cancelled_write_bytes" not in line:
                            pid_io["write"] = int(line.split("write_bytes:")[-1].strip())
                    pid_io["name"] = open("/proc/{}/comm".format(pid), "r").read().strip()
                    snapshot_second[pid] = pid_io
            except:
                if pid in snapshot_first:
                    snapshot_first.pop(pid)
                if pid in snapshot_second:
                    snapshot_second.pop(pid)

        for k, v in snapshot_first.items():
            if snapshot_first[k]["name"] == snapshot_second[k]["name"] and snapshot_first[k]["name"] != "bash":
                snapshot_read += (snapshot_second[k]["read"] - snapshot_first[k]["read"])
                snapshot_write += (snapshot_second[k]["write"] - snapshot_first[k]["write"])
        diskIO["read"] = snapshot_read
        diskIO["write"] = snapshot_write

def get_realtime_data():
    '''
    real time get system data
    :return:
    '''
    t1 = threading.Thread(
        target=_ping_thread,
        kwargs={
            'host': CU,
            'mark': '10010',
            'port': PROBEPORT
        }
    )
    t2 = threading.Thread(
        target=_ping_thread,
        kwargs={
            'host': CT,
            'mark': '189',
            'port': PROBEPORT
        }
    )
    t3 = threading.Thread(
        target=_ping_thread,
        kwargs={
            'host': CM,
            'mark': '10086',
            'port': PROBEPORT
        }
    )
    t4 = threading.Thread(
        target=_net_speed,
    )
    t5 = threading.Thread(
        target=_disk_io,
    )
    for ti in [t1, t2, t3, t4, t5]:
        ti.daemon = True
        ti.start()


def _monitor_thread(name, host, interval, type):
    while True:
        if name not in monitorServer.keys():
            break
        try:
            # 1) 解析目标 host 与端口
            if type == 'http':
                addr = str(host).replace('http://','')
                addr = addr.split('/',1)[0]
                port = 80
                if ':' in addr and not addr.startswith('['):
                    a, p = addr.rsplit(':',1)
                    if p.isdigit():
                        addr, port = a, int(p)
            elif type == 'https':
                addr = str(host).replace('https://','')
                addr = addr.split('/',1)[0]
                port = 443
                if ':' in addr and not addr.startswith('['):
                    a, p = addr.rsplit(':',1)
                    if p.isdigit():
                        addr, port = a, int(p)
            elif type == 'tcp':
                addr = str(host)
                if addr.startswith('[') and ']' in addr:
                    a = addr[1:addr.index(']')]
                    rest = addr[addr.index(']')+1:]
                    if rest.startswith(':') and rest[1:].isdigit():
                        addr, port = a, int(rest[1:])
                    else:
                        raise Exception('bad tcp target')
                else:
                    a, p = addr.rsplit(':',1)
                    addr, port = a, int(p)
            else:
                time.sleep(interval)
                continue

            # 2) 解析 IP（按偏好族）
            IP = addr
            if addr.count(':') < 1:  # 非纯 IPv6
                try:
                    if PROBE_PROTOCOL_PREFER == 'ipv4':
                        IP = socket.getaddrinfo(addr, None, socket.AF_INET)[0][4][0]
                    else:
                        IP = socket.getaddrinfo(addr, None, socket.AF_INET6)[0][4][0]
                except Exception:
                    pass

            # 3) 建连耗时（timeout=1s），ECONNREFUSED 也计入
            try:
                b = timeit.default_timer()
                socket.create_connection((IP, port), timeout=1).close()
                monitorServer[name]["latency"] = int((timeit.default_timer() - b) * 1000)
            except socket.error as error:
                if getattr(error, 'errno', None) == errno.ECONNREFUSED:
                    monitorServer[name]["latency"] = int((timeit.default_timer() - b) * 1000)
                else:
                    monitorServer[name]["latency"] = 0
        except Exception:
            monitorServer[name]["latency"] = 0
        time.sleep(interval)

def byte_str(object):
    '''
    bytes to str, str to bytes
    :param object:
    :return:
    '''
    if isinstance(object, str):
        return object.encode(encoding="utf-8")
    elif isinstance(object, bytes):
        return bytes.decode(object)
    else:
        print(type(object))

if __name__ == '__main__':
    for argc in sys.argv:
        if 'SERVER' in argc:
            SERVER = argc.split('SERVER=')[-1]
        elif 'PORT' in argc:
            PORT = int(argc.split('PORT=')[-1])
        elif 'USER' in argc:
            USER = argc.split('USER=')[-1]
        elif 'PASSWORD' in argc:
            PASSWORD = argc.split('PASSWORD=')[-1]
        elif 'INTERVAL' in argc:
            INTERVAL = int(argc.split('INTERVAL=')[-1])
    socket.setdefaulttimeout(30)
    get_realtime_data()
    while True:
        try:
            print("Connecting...")
            s = socket.create_connection((SERVER, PORT))
            data = byte_str(s.recv(1024))
            if data.find("Authentication required") > -1:
                s.send(byte_str(USER + ':' + PASSWORD + '\n'))
                data = byte_str(s.recv(1024))
                if data.find("Authentication successful") < 0:
                    print(data)
                    raise socket.error
            else:
                print(data)
                raise socket.error

            print(data)
            if data.find("You are connecting via") < 0:
                data = byte_str(s.recv(1024))
                print(data)
                monitorServer.clear()
                for i in data.split('\n'):
                    if "monitor" in i and "type" in i and "{" in i and "}" in i:
                        jdata = json.loads(i[i.find("{"):i.find("}")+1])
                        monitorServer[jdata.get("name")] = {
                            "type": jdata.get("type"),
                            "host": jdata.get("host"),
                            "latency": 0
                        }
                        t = threading.Thread(
                            target=_monitor_thread,
                            kwargs={
                                'name': jdata.get("name"),
                                'host': jdata.get("host"),
                                'interval': jdata.get("interval"),
                                'type': jdata.get("type")
                            }
                        )
                        t.daemon = True
                        t.start()

            timer = 0
            check_ip = 0
            if data.find("IPv4") > -1:
                check_ip = 6
            elif data.find("IPv6") > -1:
                check_ip = 4
            else:
                print(data)
                raise socket.error

            while True:
                CPU = get_cpu()
                NET_IN, NET_OUT = liuliang()
                Uptime = get_uptime()
                Load_1, Load_5, Load_15 = os.getloadavg()
                MemoryTotal, MemoryUsed, SwapTotal, SwapFree = get_memory()
                HDDTotal, HDDUsed = get_hdd()
                array = {}
                if not timer:
                    array['online' + str(check_ip)] = get_network(check_ip)
                    timer = 10
                else:
                    timer -= 1*INTERVAL

                array['uptime'] = Uptime
                array['load_1'] = Load_1
                array['load_5'] = Load_5
                array['load_15'] = Load_15
                array['memory_total'] = MemoryTotal
                array['memory_used'] = MemoryUsed
                array['swap_total'] = SwapTotal
                array['swap_used'] = SwapTotal - SwapFree
                array['hdd_total'] = HDDTotal
                array['hdd_used'] = HDDUsed
                array['cpu'] = CPU
                array['network_rx'] = netSpeed.get("netrx")
                array['network_tx'] = netSpeed.get("nettx")
                array['network_in'] = NET_IN
                array['network_out'] = NET_OUT
                array['ping_10010'] = lostRate.get('10010') * 100
                array['ping_189'] = lostRate.get('189') * 100
                array['ping_10086'] = lostRate.get('10086') * 100
                array['time_10010'] = pingTime.get('10010')
                array['time_189'] = pingTime.get('189')
                array['time_10086'] = pingTime.get('10086')
                array['tcp'], array['udp'], array['process'], array['thread'] = tupd()
                array['io_read'] = diskIO.get("read")
                array['io_write'] = diskIO.get("write")
                # report OS (normalized)
                try:
                    sysname = platform.system().lower()
                    if sysname.startswith('linux'):
                        os_name = 'linux'
                        # try distro from os-release
                        try:
                            with open('/etc/os-release') as f:
                                for line in f:
                                    if line.startswith('ID='):
                                        val = line.strip().split('=',1)[1].strip().strip('"')
                                        if val: os_name = val
                                        break
                        except Exception:
                            pass
                    elif sysname.startswith('darwin'):
                        os_name = 'darwin'
                    elif sysname.startswith('freebsd'):
                        os_name = 'freebsd'
                    elif sysname.startswith('openbsd'):
                        os_name = 'openbsd'
                    elif sysname.startswith('netbsd'):
                        os_name = 'netbsd'
                    else:
                        os_name = sysname or 'unknown'
                except Exception:
                    os_name = 'unknown'
                array['os'] = os_name
                items = []
                for _n, st in monitorServer.items():
                    key = str(_n)
                    try:
                        ms = int(st.get('latency') or 0)
                    except Exception:
                        ms = 0
                    items.append((key, max(0, ms)))
                # 稳定顺序：按 key 排序
                items.sort(key=lambda x: x[0])
                array['custom'] = ';'.join(f"{k}={v}" for k,v in items)
                s.send(byte_str("update " + json.dumps(array) + "\n"))
        except KeyboardInterrupt:
            raise
        except socket.error:
            monitorServer.clear()
            print("Disconnected...")
            if 's' in locals().keys():
                del s
            time.sleep(3)
        except Exception as e:
            monitorServer.clear()
            print("Caught Exception:", e)
            if 's' in locals().keys():
                del s
            time.sleep(3)
