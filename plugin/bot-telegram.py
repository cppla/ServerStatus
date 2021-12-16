#!/usr/bin/env python3
# coding: utf-8
# Create by : https://github.com/lidalao/ServerStatus
# 版本：0.0.1, 支持Python版本：2.7 to 3.9
# 支持操作系统： Linux, OSX, FreeBSD, OpenBSD and NetBSD, both 32-bit and 64-bit architectures

import os
import sys
import requests
import time
import traceback

NODE_STATUS_URL = 'http://serverstatus/json/stats.json'

offs = []
counterOff = {}
counterOn = {}

def _send(text):
    chat_id = os.getenv('TG_CHAT_ID')
    bot_token = os.environ.get('TG_BOT_TOKEN')
    url = f"https://api.telegram.org/bot{bot_token}/sendMessage?parse_mode=HTML&disable_web_page_preview=true&chat_id=" + chat_id + "&text=" + text
    try:
        requests.get(url)
    except Exception as e:
        print("catch exception: ", traceback.format_exc())

def send2tg(srv, flag):
    if srv not in counterOff:
        counterOff[srv] = 0
    if srv not in counterOn:
        counterOn[srv] = 0

    if flag == 1 : # online
        if srv in offs:
            if counterOn[srv] < 10:
                counterOn[srv] += 1
                return
            #1. Remove srv from offs; 2. Send to tg: I am online
            offs.remove(srv)
            counterOn[srv] = 0
            text = '<b>Server Status</b>' + '\n主机上线: ' + srv 
            _send(text)
    else: #offline
        if srv not in offs:
            if counterOff[srv] < 10:
                counterOff[srv] += 1
                return
            #1. Append srv to offs; 2. Send to tg: I am offline
            offs.append(srv)
            counterOff[srv] = 0
            text = '<b>Server Status</b>' + '\n主机下线: ' + srv 
            _send(text)

def sscmd(address):
    while True:
        r = requests.get(url=address, headers={"User-Agent": "ServerStatus/20211116"})
        try:
            jsonR = r.json()
        except Exception as e:
            print('未发现任何节点')
            continue
        for i in jsonR["servers"]:
            if i["online4"] is False and i["online6"] is False:
                send2tg(i["name"], 0)
            else:
                send2tg(i["name"], 1)


        time.sleep(3)

if __name__ == '__main__':
    sscmd(NODE_STATUS_URL)
