# -*- coding: utf-8 -*-
# Update by : https://github.com/cppla/ServerStatus
# 依赖于psutil跨平台库：
# 支持Python版本：2.6 to 3.5 (users of Python 2.4 and 2.5 may use 2.1.3 version)
# 支持操作系统： Linux, Windows, OSX, Sun Solaris, FreeBSD, OpenBSD and NetBSD, both 32-bit and 64-bit architectures
# 时间： 20180312

SERVER = "127.0.0.1"
PORT = 35601
USER = "s01"
PASSWORD = "USER_DEFAULT_PASSWORD"
INTERVAL = 1 # 更新间隔


import socket
import time
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
	try:
		MemUsed = Mem.total - (Mem.cached + Mem.free)
	except:
		MemUsed = Mem.total - Mem.free
	return int(Mem.total/1024.0), int(MemUsed/1024.0)

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
	for disk in disks.itervalues():
		usage = psutil.disk_usage(disk)
		size += usage.total
		used += usage.used
	return int(size/1024.0/1024.0), int(used/1024.0/1024.0)

def get_cpu():
	return psutil.cpu_percent(interval=INTERVAL)

class Traffic:
	def __init__(self):
		self.rx = collections.deque(maxlen=10)
		self.tx = collections.deque(maxlen=10)
	def get(self):
		avgrx = 0; avgtx = 0
		for name, stats in psutil.net_io_counters(pernic=True).iteritems():
			if name == "lo" or name.find("tun") > -1 \
					or name.find("docker") > -1 or name.find("veth") > -1 \
					or name.find("br-") > -1:
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

		avgrx = int(avgrx / l / INTERVAL)
		avgtx = int(avgtx / l / INTERVAL)

		return avgrx, avgtx

def liuliang():
    NET_IN = 0
    NET_OUT = 0
    net = psutil.net_io_counters(pernic=True)
    for k, v in net.items():
        if k == 'lo' or 'tun' in k \
				or 'br-' in k \
				or 'docker' in k or 'veth' in k:
            continue
        else:
            NET_IN += v[1]
            NET_OUT += v[0]
    return NET_IN, NET_OUT

def ip_status():
	object_check = ['www.10010.com', 'www.189.cn', 'www.10086.cn']
	ip_check = 0
	for i in object_check:
		s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
		s.settimeout(1)
		try:
			s.connect((i, 80))
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
		return True
	except:
		pass
	return False

lostRate = {}
def _ping_thread(host, mark):
    output = os.popen('ping -O %s &' % host if 'linux' in sys.platform else 'ping %s -t &' % host)
    lostCount = 0
    allCount = 0
    startTime = time.time()
    output.readline()
    output.readline()
    while True:
        if 'TTL' not in output.readline().upper():
            lostCount += 1
        allCount += 1
        lostRate[mark] = "%.4f" % (float(lostCount) / allCount)
        endTime = time.time()
        if endTime-startTime > 3600:
            lostCount = 0
            allCount = 0
            startTime = endTime

def get_packetLostRate():
	t1 = threading.Thread(
		target=_ping_thread,
		kwargs={
			'host': 'www.10010.com',
			'mark': '10010'
		}
	)
	t2 = threading.Thread(
		target=_ping_thread,
		kwargs={
			'host': 'www.189.cn',
			'mark': '189'
		}
	)
	t3 = threading.Thread(
		target=_ping_thread,
		kwargs={
			'host': 'bj.10086.cn',
			'mark': '10086'
		}
	)
	t1.setDaemon(True)
	t2.setDaemon(True)
	t3.setDaemon(True)
	t1.start()
	t2.start()
	t3.start()

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
    while 1:
		try:
			print("Connecting...")
			s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
			s.connect((SERVER, PORT))
			data = s.recv(1024)
			if data.find("Authentication required") > -1:
				s.send(USER + ':' + PASSWORD + '\n')
				data = s.recv(1024)
				if data.find("Authentication successful") < 0:
					print(data)
					raise socket.error
			else:
				print(data)
				raise socket.error

			print(data)
			data = s.recv(1024)
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

				s.send("update " + json.dumps(array) + "\n")
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
