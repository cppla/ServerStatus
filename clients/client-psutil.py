#!/usr/bin/env python
# coding: utf-8
# Update by : https://github.com/cppla/ServerStatus
# 依赖于psutil跨平台库
# 支持Python版本：2.7 to 3.7
# 支持操作系统： Linux, Windows, OSX, Sun Solaris, FreeBSD, OpenBSD and NetBSD, both 32-bit and 64-bit architectures
# 时间： 20200407
# 说明: 默认情况下修改server和user就可以了。丢包率监测方向可以自定义，例如：CU = "www.facebook.com"。

SERVER = "127.0.0.1"
USER = "s01"



PORT = 35601
PASSWORD = "USER_DEFAULT_PASSWORD"
INTERVAL = 1
PORBEPORT = 80
CU = "cu.tz.cloudcpp.com"
CT = "ct.tz.cloudcpp.com"
CM = "cm.tz.cloudcpp.com"

import socket
import time
import timeit
import os
import json
import collections
import psutil
import sys
import threading

def get_uptime():
    return int(time.time() - psutil.boot_time())

def get_memory():
    Mem = psutil.virtual_memory()
    return int(Mem.total / 1024.0), int(Mem.used / 1024.0)

def get_swap():
    Mem = psutil.swap_memory()
    return int(Mem.total/1024.0), int(Mem.used/1024.0)

def get_hdd():
    valid_fs = [ "ext4", "ext3", "ext2", "reiserfs", "jfs", "btrfs", "fuseblk", "zfs", "simfs", "ntfs", "fat32", "exfat", "xfs" ]
    disks = dict()
    size = 0
    used = 0
    for disk in psutil.disk_partitions():
        if not disk.device in disks and disk.fstype.lower() in valid_fs:
            disks[disk.device] = disk.mountpoint
    for disk in disks.values():
        usage = psutil.disk_usage(disk)
        size += usage.total
        used += usage.used
    return int(size/1024.0/1024.0), int(used/1024.0/1024.0)

def get_cpu():
    return psutil.cpu_percent(interval=INTERVAL)

traffic_clock = time.time()
traffic_diff = 0
def TCLOCK(func):
    def wrapper(*args, **kwargs):
        global traffic_clock, traffic_diff
        now_clock = time.time()
        traffic_diff = now_clock - traffic_clock
        traffic_clock = now_clock
        return func(*args, **kwargs)
    return wrapper

class Traffic:
    def __init__(self):
        self.rx = collections.deque(maxlen=10)
        self.tx = collections.deque(maxlen=10)

    @TCLOCK
    def get(self):
        avgrx = 0; avgtx = 0
        for name, stats in psutil.net_io_counters(pernic=True).items():
            if "lo" in name or "tun" in name \
                or "docker" in name or "veth" in name \
                or "br-" in name or "vmbr" in name \
                or "vnet" in name or "kube" in name:
                continue
            avgrx += stats.bytes_recv
            avgtx += stats.bytes_sent

        self.rx.append(avgrx)
        self.tx.append(avgtx)
        avgrx = 0; avgtx = 0

        l = len(self.rx)
        for x in range(l - 1):
            avgrx += self.rx[x+1] - self.rx[x]
            avgtx += self.tx[x+1] - self.tx[x]

        avgrx = int(avgrx / l / traffic_diff)
        avgtx = int(avgtx / l / traffic_diff)

        return avgrx, avgtx

def liuliang():
    NET_IN = 0
    NET_OUT = 0
    net = psutil.net_io_counters(pernic=True)
    for k, v in net.items():
        if 'lo' in k or 'tun' in k \
                or 'docker' in k or 'veth' in k \
                or 'br-' in k or 'vmbr' in k \
                or 'vnet' in k or 'kube' in k:
            continue
        else:
            NET_IN += v[1]
            NET_OUT += v[0]
    return NET_IN, NET_OUT

def tupd():
    '''
    tcp, udp, process, thread count: for view ddcc attack , then send warning
    :return:
    '''
    if 'linux' in sys.platform:
        t = int(os.popen('ss -t|wc -l').read()[:-1])-1
        u = int(os.popen('ss -u|wc -l').read()[:-1])-1
        p = int(os.popen('ps -ef|wc -l').read()[:-1])-2
        d = int(os.popen('ps -eLf|wc -l').read()[:-1])-2
    else:
        t = int(os.popen('netstat -an|find "TCP" /c').read()[:-1])-1
        u = int(os.popen('netstat -an|find "UDP" /c').read()[:-1])-1
        p = len(psutil.pids())
        d = 0
        # cpu is high, default: 0
        # d = sum([psutil.Process(k).num_threads() for k in [x for x in psutil.pids()]])
    return t,u,p,d

def ip_status():
    ip_check = 0
    for i in [CU, CT, CM]:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(1)
        try:
            s.connect((i, PORBEPORT))
        except:
            ip_check += 1
        s.close()
        del s
    if ip_check >= 2:
        return False
    else:
        return True

def get_network(ip_version):
    if(ip_version == 4):
        HOST = "ipv4.google.com"
    elif(ip_version == 6):
        HOST = "ipv6.google.com"
    try:
        s = socket.create_connection((HOST, 80), 2)
        s.close()
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
def _ping_thread(host, mark, port):
    lostPacket = 0
    allPacket = 0
    startTime = time.time()

    while True:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(1)
        try:
            b = timeit.default_timer()
            s.connect((host, port))
            pingTime[mark] = int((timeit.default_timer() - b) * 1000)
        except:
            lostPacket += 1
        finally:
            allPacket += 1
        s.close()

        if allPacket > 100:
            lostRate[mark] = float(lostPacket) / allPacket

        endTime = time.time()
        if endTime - startTime > 3600:
            lostPacket = 0
            allPacket = 0
            startTime = endTime

        time.sleep(1)

def get_packetLostRate():
    t1 = threading.Thread(
        target=_ping_thread,
        kwargs={
            'host': CU,
            'mark': '10010',
            'port': PORBEPORT
        }
    )
    t2 = threading.Thread(
        target=_ping_thread,
        kwargs={
            'host': CT,
            'mark': '189',
            'port': PORBEPORT
        }
    )
    t3 = threading.Thread(
        target=_ping_thread,
        kwargs={
            'host': CM,
            'mark': '10086',
            'port': PORBEPORT
        }
    )
    t1.setDaemon(True)
    t2.setDaemon(True)
    t3.setDaemon(True)
    t1.start()
    t2.start()
    t3.start()

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
    get_packetLostRate()
    while 1:
        try:
            print("Connecting...")
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.connect((SERVER, PORT))
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
            data = byte_str(s.recv(1024))
            print(data)

            timer = 0
            check_ip = 0
            if data.find("IPv4") > -1:
                check_ip = 6
            elif data.find("IPv6") > -1:
                check_ip = 4
            else:
                print(data)
                raise socket.error

            traffic = Traffic()
            traffic.get()
            while 1:
                CPU = get_cpu()
                NetRx, NetTx = traffic.get()
                NET_IN, NET_OUT = liuliang()
                Uptime = get_uptime()
                Load_1, Load_5, Load_15 = os.getloadavg() if 'linux' in sys.platform else (0.0, 0.0, 0.0)
                MemoryTotal, MemoryUsed = get_memory()
                SwapTotal, SwapUsed = get_swap()
                HDDTotal, HDDUsed = get_hdd()
                IP_STATUS = ip_status()

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
                array['swap_used'] = SwapUsed
                array['hdd_total'] = HDDTotal
                array['hdd_used'] = HDDUsed
                array['cpu'] = CPU
                array['network_rx'] = NetRx
                array['network_tx'] = NetTx
                array['network_in'] = NET_IN
                array['network_out'] = NET_OUT
                array['ip_status'] = IP_STATUS
                array['ping_10010'] = lostRate.get('10010') * 100
                array['ping_189'] = lostRate.get('189') * 100
                array['ping_10086'] = lostRate.get('10086') * 100
                array['time_10010'] = pingTime.get('10010')
                array['time_189'] = pingTime.get('189')
                array['time_10086'] = pingTime.get('10086')
                array['tcp'], array['udp'], array['process'], array['thread'] = tupd()

                s.send(byte_str("update " + json.dumps(array) + "\n"))
        except KeyboardInterrupt:
            raise
        except socket.error:
            print("Disconnected...")
            # keep on trying after a disconnect
            s.close()
            time.sleep(3)
        except Exception as e:
            print("Caught Exception:", e)
            s.close()
            time.sleep(3)
