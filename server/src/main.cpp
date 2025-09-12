#define __STDC_FORMAT_MACROS
#include <inttypes.h>
#include <time.h>
#include <detect.h>
#include <system.h>
#include <argparse.h>
#include <json.h>
#include "server.h"
#include "main.h"
#include "exprtk.hpp"
#include "curl/curl.h"
#include <stdio.h>
#include <stdlib.h>
#include <string>

// 全局运行标志（需在 SSLCheckThread 定义前初始化）
static volatile int gs_Running = 1;
static volatile int gs_ReloadConfig = 0;

static int64_t ParseOpenSSLEnddate(const char *line)
{
	// line format: notAfter=Aug 12 23:59:59 2025 GMT
	const char *p = strstr(line, "notAfter=");
	if(!p) return 0;
	p += 9;
	struct tm tmv; memset(&tmv,0,sizeof(tmv));
	char month[4]={0};
	int day, hour, min, sec, year;
	if(sscanf(p, "%3s %d %d:%d:%d %d GMT", month, &day, &hour, &min, &sec, &year)!=6) return 0;
	const char *months="JanFebMarAprMayJunJulAugSepOctNovDec";
	const char *mpos = strstr(months, month);
	if(!mpos) return 0;
	int mon = (int)((mpos - months)/3);
	tmv.tm_year = year - 1900;
	tmv.tm_mon = mon;
	tmv.tm_mday = day;
	tmv.tm_hour = hour; tmv.tm_min = min; tmv.tm_sec = sec;
	time_t t = timegm(&tmv);
	return (int64_t)t;
}

struct SSLCheckThreadData { CMain *pMain; };
static void SSLCheckThread(void *pUser)
{
	SSLCheckThreadData *pData = (SSLCheckThreadData*)pUser;
	while(gs_Running){
		for(int i=0;i<NET_MAX_CLIENTS;i++){
			if(!pData->pMain->SSLCert(i) || !strcmp(pData->pMain->SSLCert(i)->m_aName, "NULL")) break;
			CMain::CSSLCerts *cert = pData->pMain->SSLCert(i);
			time_t nowt = time(0);
			if(cert->m_aLastCheck !=0 && (nowt - cert->m_aLastCheck) < cert->m_aInterval) continue;
			cert->m_aLastCheck = nowt;
			char cmd[1024];
			// 说明: 通过 s_client 获取证书，再用 x509 解析到期时间；统一屏蔽 stderr 以防握手失败/非 TLS 端口时刷屏。
			// 若配置中写成 https://domain/path 则需要清洗。
			char cleanHost[256];
			str_copy(cleanHost, cert->m_aDomain, sizeof(cleanHost));
			// 去协议
			if(!strncasecmp(cleanHost, "https://", 8)) memmove(cleanHost, cleanHost+8, strlen(cleanHost+8)+1);
			else if(!strncasecmp(cleanHost, "http://", 7)) memmove(cleanHost, cleanHost+7, strlen(cleanHost+7)+1);
			// 去路径
			char *slash = strchr(cleanHost, '/'); if(slash) *slash='\0';
			// 若含 :port 再截取主机部分（端口由配置提供）
			char *colon = strchr(cleanHost, ':'); if(colon) *colon='\0';
			int n = snprintf(cmd,sizeof(cmd),"echo | openssl s_client -servername %s -connect %s:%d </dev/null 2>/dev/null | openssl x509 -noout -enddate -text 2>/dev/null", cleanHost, cleanHost, cert->m_aPort);
			if(n <= 0 || n >= (int)sizeof(cmd)) continue; // 避免截断执行
			FILE *fp = popen(cmd, "r");
			if(!fp) continue;
			char line[1024]={0};
			int foundEnddate=0;
			int mismatch = 1; // 默认视为不匹配，发现任一匹配域名再置0
			int haveNames = 0;
			// 将目标域名转为小写
			char target[256]; str_copy(target, cleanHost, sizeof(target));
			for(char *p=target; *p; ++p) *p=tolower(*p);
			while(fgets(line,sizeof(line),fp)){
				if(!foundEnddate){
					int64_t expire = ParseOpenSSLEnddate(line);
					if(expire>0){ cert->m_aExpireTS = expire; foundEnddate=1; }
				}
				// 解析 subjectAltName
				// 解析 Subject 中的 CN（备用）
				char *subj = strstr(line, "Subject:");
				if(subj){
					char *cn = strstr(subj, " CN=");
					if(cn){
						cn += 4; // 跳过 ' CN='
						char name[256]={0}; int ni=0;
						while(*cn && *cn!='/' && *cn!=',' && *cn!='\n' && ni<(int)sizeof(name)-1){ name[ni++]=*cn++; }
						name[ni]='\0';
						while(ni>0 && (name[ni-1]==' '||name[ni-1]=='\r'||name[ni-1]=='\t')){ name[--ni]='\0'; }
						for(char *q=name; *q; ++q) *q=tolower(*q);
						if(ni>0){
							haveNames=1;
							int match=0;
							if(name[0]=='*' && name[1]=='.'){
								const char *sub = strchr(target,'.');
								if(sub && !strcmp(sub+1, name+2)) match=1;
							}else if(!strcmp(name,target)) match=1;
							if(match){ mismatch=0; }
						}
					}
				}
				if(strstr(line, "DNS:")){
					char *p = line;
					while((p = strstr(p, "DNS:"))){
						p += 4; while(*p==' '){p++;}
						char name[256]={0}; int ni=0;
						while(*p && *p!=',' && *p!='\n' && ni<(int)sizeof(name)-1){ name[ni++]=*p++; }
						name[ni]='\0';
						// 去空白
						while(ni>0 && (name[ni-1]==' '||name[ni-1]=='\r'||name[ni-1]=='\t')){ name[--ni]='\0'; }
						for(char *q=name; *q; ++q) *q=tolower(*q);
						haveNames=1;
						// 通配符匹配 *.example.com
						int match=0;
						if(name[0]=='*' && name[1]=='.'){
							const char *sub = strchr(target,'.');
							if(sub && !strcmp(sub+1, name+2)) match=1;
						}else if(!strcmp(name,target)) match=1;
						if(match){ mismatch=0; goto names_done; }
					}
				}
			}
names_done:
			pclose(fp);
			if(haveNames){ cert->m_aHostnameMismatch = mismatch ? 1 : 0; }
			else { /* 未能提取任何域名，保留原状态，不触发误报 */ }
			// 告警: 仅在不匹配且 24h 冷却
			if(cert->m_aHostnameMismatch==1){
				if(cert->m_aLastAlarmMismatch==0 || nowt - cert->m_aLastAlarmMismatch > 24*3600){
					if(strlen(cert->m_aCallback)>0){
						CURL *curl = curl_easy_init();
						if(curl){
							char msg[1024];
							snprintf(msg,sizeof(msg),"【SSL证书域名不匹配】%s(%s) 证书域名与配置不一致", cert->m_aName, cert->m_aDomain);
							char *enc = curl_easy_escape(curl,msg,0);
							char url[1500]; snprintf(url,sizeof(url),"%s%s", cert->m_aCallback, enc?enc:"");
							curl_easy_setopt(curl, CURLOPT_POST, 1L);
							curl_easy_setopt(curl, CURLOPT_URL, url);
							curl_easy_setopt(curl, CURLOPT_POSTFIELDS, "signature=ServerStatusSSL");
							curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0);
							curl_easy_setopt(curl, CURLOPT_SSL_VERIFYHOST, 0);
							curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 3L);
							curl_easy_setopt(curl, CURLOPT_TIMEOUT, 6L);
							curl_easy_perform(curl);
							if(enc) curl_free(enc);
							curl_easy_cleanup(curl);
						}
					}
					cert->m_aLastAlarmMismatch = nowt;
				}
			}
			// alarm logic
			if(cert->m_aExpireTS>0){
				// 实际剩余天数（向上取整），用于提醒文本展示
				int64_t secsLeft = cert->m_aExpireTS - nowt;
				int days = (int)(secsLeft/86400); // 分桶用：向下取整
				int daysLeft = secsLeft>0 ? (int)((secsLeft + 86399)/86400) : 0; // 文案用：向上取整
				int64_t *lastAlarm = NULL; int need=0; int target=0;
				if(days <=7 && days >3){ lastAlarm=&cert->m_aLastAlarm7; target=7; }
				else if(days <=3 && days >1){ lastAlarm=&cert->m_aLastAlarm3; target=3; }
				else if(days <=1){ lastAlarm=&cert->m_aLastAlarm1; target=1; }
				if(lastAlarm && (*lastAlarm==0 || nowt - *lastAlarm > 20*3600)) need=1; // avoid spam, 20h
				if(need && strlen(cert->m_aCallback)>0){
					CURL *curl = curl_easy_init();
					if(curl){
						char msg[1024];
						char timebuf[32];
						time_t expt = (time_t)cert->m_aExpireTS;
						strftime(timebuf,sizeof(timebuf),"%Y-%m-%d %H:%M:%S", gmtime(&expt));
						// 统一策略：与 JSON 中 expire_days 一致，使用 floor 结果 days
										snprintf(msg,sizeof(msg),"【SSL证书提醒】%s(%s) 将在 %d 天后(%s UTC) 到期", cert->m_aName, cert->m_aDomain, days, timebuf);
						char *enc = curl_easy_escape(curl,msg,0);
						char url[1500]; snprintf(url,sizeof(url),"%s%s", cert->m_aCallback, enc?enc:"");
						curl_easy_setopt(curl, CURLOPT_POST, 1L);
						curl_easy_setopt(curl, CURLOPT_URL, url);
						curl_easy_setopt(curl, CURLOPT_POSTFIELDS, "signature=ServerStatusSSL");
						curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0);
						curl_easy_setopt(curl, CURLOPT_SSL_VERIFYHOST, 0);
						curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 3L);
						curl_easy_setopt(curl, CURLOPT_TIMEOUT, 6L);
						curl_easy_perform(curl);
						if(enc) curl_free(enc);
						curl_easy_cleanup(curl);
					}
					*lastAlarm = nowt;
				}
			}
		}
		thread_sleep(5000);
	}
}

#if defined(CONF_FAMILY_UNIX)
	#include <signal.h>
#endif

#ifndef PRId64
	#define PRId64 "I64d"
#endif

static void ExitFunc(int Signal)
{
	printf("[EXIT] Caught signal %d\n", Signal);
	gs_Running = 0;
}

static void ReloadFunc(int Signal)
{
	printf("[RELOAD] Caught signal %d\n", Signal);
	gs_ReloadConfig = 1;
}

CConfig::CConfig()
{
	// Initialize to default values
	m_Verbose = false; // -v, --verbose
	str_copy(m_aConfigFile, "config.json", sizeof(m_aConfigFile)); // -c, --config
	str_copy(m_aWebDir, "../web/", sizeof(m_aJSONFile)); // -d, --web-dir
	str_copy(m_aTemplateFile, "template.html", sizeof(m_aTemplateFile));
	str_copy(m_aJSONFile, "json/stats.json", sizeof(m_aJSONFile));
	str_copy(m_aBindAddr, "", sizeof(m_aBindAddr)); // -b, --bind
	m_Port = 35601; // -p, --port
}

CMain::CMain(CConfig Config) : m_Config(Config)
{
	mem_zero(m_aClients, sizeof(m_aClients));
	for(int i = 0; i < NET_MAX_CLIENTS; i++)
		m_aClients[i].m_ClientNetID = -1;
}

CMain::CClient *CMain::ClientNet(int ClientNetID)
{
	if(ClientNetID < 0 || ClientNetID >= NET_MAX_CLIENTS)
		return 0;

	for(int i = 0; i < NET_MAX_CLIENTS; i++)
	{
		if(Client(i)->m_ClientNetID == ClientNetID)
			return Client(i);
	}

	return 0;
}

int CMain::ClientNetToClient(int ClientNetID)
{
	if(ClientNetID < 0 || ClientNetID >= NET_MAX_CLIENTS)
		return -1;

	for(int i = 0; i < NET_MAX_CLIENTS; i++)
	{
		if(Client(i)->m_ClientNetID == ClientNetID)
			return i;
	}

	return -1;
}

void CMain::OnNewClient(int ClientNetID, int ClientID)
{
	dbg_msg("main", "OnNewClient(ncid=%d, cid=%d)", ClientNetID, ClientID);
	Client(ClientID)->m_ClientNetID = ClientNetID;
	Client(ClientID)->m_ClientNetType = m_Server.Network()->ClientAddr(ClientNetID)->type;
	Client(ClientID)->m_TimeConnected = time_get();
	Client(ClientID)->m_Connected = true;

	if(Client(ClientID)->m_ClientNetType == NETTYPE_IPV4)
		Client(ClientID)->m_Stats.m_Online4 = true;
	else if(Client(ClientID)->m_ClientNetType == NETTYPE_IPV6)
		Client(ClientID)->m_Stats.m_Online6 = true;

    // Send monitor to client
    // support by cpp.la
    int ID = 0;
    char monitorBuffer[2048];
    while (strcmp(Monitors(ID)->m_aName, "NULL"))
    {
        memset(monitorBuffer, 0, sizeof(monitorBuffer));
        sprintf(monitorBuffer, "{\"name\":\"%s\",\"host\":\"%s\",\"interval\":%d,\"type\":\"%s\",\"monitor\":%d}", Monitors(ID)->m_aName, Monitors(ID)->m_aHost, Monitors(ID)->m_aInterval, Monitors(ID)->m_aType, ID);
        m_Server.Network()->Send(ClientNetID, monitorBuffer);
        ID++;
    }
}

void CMain::OnDelClient(int ClientNetID)
{
	int ClientID = ClientNetToClient(ClientNetID);
	dbg_msg("main", "OnDelClient(ncid=%d, cid=%d)", ClientNetID, ClientID);
	if(ClientID >= 0 && ClientID < NET_MAX_CLIENTS)
	{
		Client(ClientID)->m_Connected = false;
		Client(ClientID)->m_ClientNetID = -1;
		Client(ClientID)->m_ClientNetType = NETTYPE_INVALID;
		mem_zero(&Client(ClientID)->m_Stats, sizeof(CClient::CStats));
	}
    m_OfflineAlarmThreadData.pClients = m_aClients;
    m_OfflineAlarmThreadData.pWatchDogs = m_aCWatchDogs;
    m_OfflineAlarmThreadData.m_ReloadRequired = ClientID;
    thread_create(offlineAlarmThread, &m_OfflineAlarmThreadData);
}

int CMain::HandleMessage(int ClientNetID, char *pMessage)
{
	CClient *pClient = ClientNet(ClientNetID);
	if(!pClient)
		return true;

	if(str_comp_num(pMessage, "update", sizeof("update")-1) == 0)
	{
		char *pData = str_skip_whitespaces(&pMessage[sizeof("update")-1]);

		// parse json data
		json_settings JsonSettings;
		mem_zero(&JsonSettings, sizeof(JsonSettings));
		char aError[256];
		json_value *pJsonData = json_parse_ex(&JsonSettings, pData, strlen(pData), aError);
		if(!pJsonData)
		{
			dbg_msg("main", "JSON Error: %s", aError);
			if(pClient->m_Stats.m_Pong)
				m_Server.Network()->Send(ClientNetID, "1");
			return 1;
		}

		// extract data
		const json_value &rStart = (*pJsonData);
		if(rStart["uptime"].type)
			pClient->m_Stats.m_Uptime = rStart["uptime"].u.integer;
		if(rStart["load_1"].type)
			pClient->m_Stats.m_Load_1 = rStart["load_1"].u.dbl;
		if(rStart["load_5"].type)
			pClient->m_Stats.m_Load_5 = rStart["load_5"].u.dbl;
		if(rStart["load_15"].type)
			pClient->m_Stats.m_Load_15 = rStart["load_15"].u.dbl;
		if(rStart["ping_10010"].type)
			pClient->m_Stats.m_ping_10010 = rStart["ping_10010"].u.dbl;
		if(rStart["ping_189"].type)
			pClient->m_Stats.m_ping_189 = rStart["ping_189"].u.dbl;
		if(rStart["ping_10086"].type)
			pClient->m_Stats.m_ping_10086 = rStart["ping_10086"].u.dbl;
	    if(rStart["time_10010"].type)
			pClient->m_Stats.m_time_10010 = rStart["time_10010"].u.integer;
		if(rStart["time_189"].type)
			pClient->m_Stats.m_time_189 = rStart["time_189"].u.integer;
		if(rStart["time_10086"].type)
			pClient->m_Stats.m_time_10086 = rStart["time_10086"].u.integer;
		if(rStart["tcp"].type)
			pClient->m_Stats.m_tcpCount = rStart["tcp"].u.integer;
		if(rStart["udp"].type)
			pClient->m_Stats.m_udpCount = rStart["udp"].u.integer;
		if(rStart["process"].type)
			pClient->m_Stats.m_processCount = rStart["process"].u.integer;
		if(rStart["thread"].type)
			pClient->m_Stats.m_threadCount = rStart["thread"].u.integer;
		if(rStart["network_rx"].type)
			pClient->m_Stats.m_NetworkRx = rStart["network_rx"].u.integer;
		if(rStart["network_tx"].type)
			pClient->m_Stats.m_NetworkTx = rStart["network_tx"].u.integer;
		if(rStart["network_in"].type)
			pClient->m_Stats.m_NetworkIN = rStart["network_in"].u.integer;
		if(rStart["network_out"].type)
			pClient->m_Stats.m_NetworkOUT = rStart["network_out"].u.integer;
		if(rStart["memory_total"].type)
			pClient->m_Stats.m_MemTotal = rStart["memory_total"].u.integer;
		if(rStart["memory_used"].type)
			pClient->m_Stats.m_MemUsed = rStart["memory_used"].u.integer;
		if(rStart["swap_total"].type)
			pClient->m_Stats.m_SwapTotal = rStart["swap_total"].u.integer;
		if(rStart["swap_used"].type)
			pClient->m_Stats.m_SwapUsed = rStart["swap_used"].u.integer;
		if(rStart["hdd_total"].type)
			pClient->m_Stats.m_HDDTotal = rStart["hdd_total"].u.integer;
		if(rStart["hdd_used"].type)
			pClient->m_Stats.m_HDDUsed = rStart["hdd_used"].u.integer;
		if(rStart["io_read"].type)
			pClient->m_Stats.m_IORead = rStart["io_read"].u.integer;
		if(rStart["io_write"].type)
			pClient->m_Stats.m_IOWrite = rStart["io_write"].u.integer;
		if(rStart["cpu"].type)
			pClient->m_Stats.m_CPU = rStart["cpu"].u.dbl;
		if(rStart["online4"].type && pClient->m_ClientNetType == NETTYPE_IPV6)
			pClient->m_Stats.m_Online4 = rStart["online4"].u.boolean;
		if(rStart["online6"].type && pClient->m_ClientNetType == NETTYPE_IPV4)
			pClient->m_Stats.m_Online6 = rStart["online6"].u.boolean;
		if(rStart["custom"].type == json_string)
			str_copy(pClient->m_Stats.m_aCustom, rStart["custom"].u.string.ptr, sizeof(pClient->m_Stats.m_aCustom));
		// optional OS field from clients
		if(rStart["os"].type == json_string)
			str_copy(pClient->m_Stats.m_aOS, rStart["os"].u.string.ptr, sizeof(pClient->m_Stats.m_aOS));

		//copy message for watchdog to analysis
        WatchdogMessage(ClientNetID,
                        pClient->m_Stats.m_Load_1, pClient->m_Stats.m_Load_5, pClient->m_Stats.m_Load_15,
                        pClient->m_Stats.m_ping_10010, pClient->m_Stats.m_ping_189, pClient->m_Stats.m_ping_10086,
                        pClient->m_Stats.m_time_10010, pClient->m_Stats.m_time_189, pClient->m_Stats.m_time_10086,
                        pClient->m_Stats.m_tcpCount, pClient->m_Stats.m_udpCount, pClient->m_Stats.m_processCount,
                        pClient->m_Stats.m_threadCount, pClient->m_Stats.m_NetworkRx, pClient->m_Stats.m_NetworkTx,
                        pClient->m_Stats.m_NetworkIN, pClient->m_Stats.m_NetworkOUT,
                        pClient->m_LastNetworkIN, pClient->m_LastNetworkOUT, pClient->m_Stats.m_MemTotal,
                        pClient->m_Stats.m_MemUsed, pClient->m_Stats.m_SwapTotal, pClient->m_Stats.m_SwapUsed,
                        pClient->m_Stats.m_HDDTotal, pClient->m_Stats.m_HDDUsed, pClient->m_Stats.m_IORead,
                        pClient->m_Stats.m_IOWrite, pClient->m_Stats.m_CPU, pClient->m_Stats.m_Online4,
                        pClient->m_Stats.m_Online6);

		if(m_Config.m_Verbose)
		{
			if(rStart["online4"].type)
				dbg_msg("main", "Online4: %s\nUptime: %" PRId64 "\nLoad_1: %f\nLoad_5: %f\nLoad_15: %f\nPing_10010: %f\nPing_189: %f\nPing_10086: %f\nTime_10010: %" PRId64 "\nTime_189: %" PRId64 "\nTime_10086: %" PRId64 "\nTcp_count: %" PRId64 "\nUdp_count: %" PRId64 "\nprocess_count: %" PRId64 "\nthread_count: %" PRId64 "\nNetworkRx: %" PRId64 "\nNetworkTx: %" PRId64 "\nNetworkIN: %" PRId64 "\nNetworkOUT: %" PRId64 "\nMemTotal: %" PRId64 "\nMemUsed: %" PRId64 "\nSwapTotal: %" PRId64 "\nSwapUsed: %" PRId64 "\nHDDTotal: %" PRId64 "\nHDDUsed: %" PRId64 "\nCPU: %f\nIORead: %" PRId64 "\nIOWrite: %" PRId64 "\n",
					rStart["online4"].u.boolean ? "true" : "false",
					pClient->m_Stats.m_Uptime,
					pClient->m_Stats.m_Load_1, pClient->m_Stats.m_Load_5, pClient->m_Stats.m_Load_15, pClient->m_Stats.m_ping_10010, pClient->m_Stats.m_ping_189, pClient->m_Stats.m_ping_10086, pClient->m_Stats.m_time_10010, pClient->m_Stats.m_time_189, pClient->m_Stats.m_time_10086,pClient->m_Stats.m_tcpCount,pClient->m_Stats.m_udpCount,pClient->m_Stats.m_processCount,pClient->m_Stats.m_threadCount,pClient->m_Stats.m_NetworkRx, pClient->m_Stats.m_NetworkTx, pClient->m_Stats.m_NetworkIN, pClient->m_Stats.m_NetworkOUT, pClient->m_Stats.m_MemTotal, pClient->m_Stats.m_MemUsed, pClient->m_Stats.m_SwapTotal, pClient->m_Stats.m_SwapUsed, pClient->m_Stats.m_HDDTotal, pClient->m_Stats.m_HDDUsed, pClient->m_Stats.m_CPU, pClient->m_Stats.m_IORead, pClient->m_Stats.m_IOWrite);
			else if(rStart["online6"].type)
				dbg_msg("main", "Online6: %s\nUptime: %" PRId64 "\nLoad_1: %f\nLoad_5: %f\nLoad_15: %f\nPing_10010: %f\nPing_189: %f\nPing_10086: %f\nTime_10010: %" PRId64 "\nTime_189: %" PRId64 "\nTime_10086: %" PRId64 "\nTcp_count: %" PRId64 "\nUdp_count: %" PRId64 "\nprocess_count: %" PRId64 "\nthread_count: %" PRId64 "\nNetworkRx: %" PRId64 "\nNetworkTx: %" PRId64 "\nNetworkIN: %" PRId64 "\nNetworkOUT: %" PRId64 "\nMemTotal: %" PRId64 "\nMemUsed: %" PRId64 "\nSwapTotal: %" PRId64 "\nSwapUsed: %" PRId64 "\nHDDTotal: %" PRId64 "\nHDDUsed: %" PRId64 "\nCPU: %f\nIORead: %" PRId64 "\nIOWrite: %" PRId64 "\n",
					rStart["online6"].u.boolean ? "true" : "false",
					pClient->m_Stats.m_Uptime,
					pClient->m_Stats.m_Load_1, pClient->m_Stats.m_Load_5, pClient->m_Stats.m_Load_15, pClient->m_Stats.m_ping_10010, pClient->m_Stats.m_ping_189, pClient->m_Stats.m_ping_10086, pClient->m_Stats.m_time_10010, pClient->m_Stats.m_time_189, pClient->m_Stats.m_time_10086,pClient->m_Stats.m_tcpCount,pClient->m_Stats.m_udpCount,pClient->m_Stats.m_processCount,pClient->m_Stats.m_threadCount,pClient->m_Stats.m_NetworkRx, pClient->m_Stats.m_NetworkTx, pClient->m_Stats.m_NetworkIN, pClient->m_Stats.m_NetworkOUT, pClient->m_Stats.m_MemTotal, pClient->m_Stats.m_MemUsed, pClient->m_Stats.m_SwapTotal, pClient->m_Stats.m_SwapUsed, pClient->m_Stats.m_HDDTotal, pClient->m_Stats.m_HDDUsed, pClient->m_Stats.m_CPU, pClient->m_Stats.m_IORead, pClient->m_Stats.m_IOWrite);
			else
				dbg_msg("main", "Uptime: %" PRId64 "\nLoad_1: %f\nLoad_5: %f\nLoad_15: %f\nPing_10010: %f\nPing_189: %f\nPing_10086: %f\nTime_10010: %" PRId64 "\nTime_189: %" PRId64 "\nTime_10086: %" PRId64 "\nTcp_count: %" PRId64 "\nUdp_count: %" PRId64 "\nprocess_count: %" PRId64 "\nthread_count: %" PRId64 "\nNetworkRx: %" PRId64 "\nNetworkTx: %" PRId64 "\nNetworkIN: %" PRId64 "\nNetworkOUT: %" PRId64 "\nMemTotal: %" PRId64 "\nMemUsed: %" PRId64 "\nSwapTotal: %" PRId64 "\nSwapUsed: %" PRId64 "\nHDDTotal: %" PRId64 "\nHDDUsed: %" PRId64 "\nCPU: %f\nIORead: %" PRId64 "\nIOWrite: %" PRId64 "\n",
					pClient->m_Stats.m_Uptime,
					pClient->m_Stats.m_Load_1, pClient->m_Stats.m_Load_5, pClient->m_Stats.m_Load_15, pClient->m_Stats.m_ping_10010, pClient->m_Stats.m_ping_189, pClient->m_Stats.m_ping_10086, pClient->m_Stats.m_time_10010, pClient->m_Stats.m_time_189, pClient->m_Stats.m_time_10086,pClient->m_Stats.m_tcpCount,pClient->m_Stats.m_udpCount,pClient->m_Stats.m_processCount,pClient->m_Stats.m_threadCount,pClient->m_Stats.m_NetworkRx, pClient->m_Stats.m_NetworkTx, pClient->m_Stats.m_NetworkIN, pClient->m_Stats.m_NetworkOUT, pClient->m_Stats.m_MemTotal, pClient->m_Stats.m_MemUsed, pClient->m_Stats.m_SwapTotal, pClient->m_Stats.m_SwapUsed, pClient->m_Stats.m_HDDTotal, pClient->m_Stats.m_HDDUsed, pClient->m_Stats.m_CPU, pClient->m_Stats.m_IORead, pClient->m_Stats.m_IOWrite);
		}

		// clean up
		json_value_free(pJsonData);

		if(pClient->m_Stats.m_Pong)
			m_Server.Network()->Send(ClientNetID, "0");
		return 0;
	}
	else if(str_comp_num(pMessage, "pong", sizeof("pong")-1) == 0)
	{
		char *pData = str_skip_whitespaces(&pMessage[sizeof("pong")-1]);

		if(!str_comp(pData, "0") || !str_comp(pData, "off"))
			pClient->m_Stats.m_Pong = false;
		else if(!str_comp(pData, "1") || !str_comp(pData, "on"))
			pClient->m_Stats.m_Pong = true;

		return 0;
	}

	if(pClient->m_Stats.m_Pong)
		m_Server.Network()->Send(ClientNetID, "1");

	return 1;
}


void CMain::WatchdogMessage(int ClientNetID, double load_1, double load_5, double load_15, double ping_10010, double ping_189, double ping_10086,
                            double time_10010, double time_189, double time_10086, double tcp_count, double udp_count, double process_count, double thread_count,
                            double network_rx, double network_tx, double network_in, double network_out, double last_network_in, double last_network_out, double memory_total, double memory_used,
                            double swap_total, double swap_used, double hdd_total, double hdd_used, double io_read, double io_write, double cpu,
                            double online4, double online6)
{
    int ID = 0;
    while (strcmp(Watchdog(ID)->m_aName, "NULL"))
    {
//        Exprtk库默认使用窄字符类型，但可能会出现中文等Unicode字符无法正确解析的问题。
//        todo: 为解决此问题，可以使用宽字符类型替换Exprtk库中默认的窄字符类型。
//        #include <string>
//        #include <vector>
//        #include <exprtk.hpp>
//        typedef exprtk::expression<wchar_t> expression_type;
//        typedef exprtk::parser<wchar_t> parser_type;
//        int main()
//      {
//                std::wstring expression_string = L"sin(x)";
//                expression_type expression;
//                parser_type parser;
//                parser.compile(expression_string, expression);
//                double x = 3.14;
//                double result = expression.value();
//                return 0;
//       }
        typedef exprtk::symbol_table<double> symbol_table_t;
        typedef exprtk::expression<double>   expression_t;
        typedef exprtk::parser<double>       parser_t;
        const std::string expression_string = Watchdog(ID)->m_aRule;
		int ClientID = ClientNetToClient(ClientNetID);
		if(ClientID < 0 || ClientID >= NET_MAX_CLIENTS) {
			ID++;
			continue; // 无效客户端，跳过当前 watchdog 规则
		}
		std::string username = Client(ClientID)->m_aUsername;
		std::string name = Client(ClientID)->m_aName;
		std::string type = Client(ClientID)->m_aType;
		std::string host = Client(ClientID)->m_aHost;
		std::string location = Client(ClientID)->m_aLocation;

        symbol_table_t symbol_table;
        symbol_table.add_stringvar("username", username);
        symbol_table.add_stringvar("name", name);
        symbol_table.add_stringvar("type", type);
        symbol_table.add_stringvar("host", host);
        symbol_table.add_stringvar("location", location);
        symbol_table.add_variable("load_1",load_1);
        symbol_table.add_variable("load_5",load_5);
        symbol_table.add_variable("load_15",load_15);
        symbol_table.add_variable("ping_10010",ping_10010);
        symbol_table.add_variable("ping_189",ping_189);
        symbol_table.add_variable("ping_10086",ping_10086);
        symbol_table.add_variable("time_10010",time_10010);
        symbol_table.add_variable("time_189",time_189);
        symbol_table.add_variable("time_10086",time_10086);
        symbol_table.add_variable("tcp_count",tcp_count);
        symbol_table.add_variable("udp_count",udp_count);
        symbol_table.add_variable("process_count",process_count);
        symbol_table.add_variable("thread_count",thread_count);
        symbol_table.add_variable("network_rx",network_rx);
        symbol_table.add_variable("network_tx",network_tx);
        symbol_table.add_variable("network_in",network_in);
        symbol_table.add_variable("network_out",network_out);
        symbol_table.add_variable("last_network_in",last_network_in);
        symbol_table.add_variable("last_network_out",last_network_out);
        symbol_table.add_variable("memory_total",memory_total);
        symbol_table.add_variable("memory_used",memory_used);
        symbol_table.add_variable("swap_total",swap_total);
        symbol_table.add_variable("swap_used",swap_used);
        symbol_table.add_variable("hdd_total",hdd_total);
        symbol_table.add_variable("hdd_used",hdd_used);
        symbol_table.add_variable("io_read",io_read);
        symbol_table.add_variable("io_write",io_write);
        symbol_table.add_variable("cpu",cpu);
        symbol_table.add_variable("online4",online4);
        symbol_table.add_variable("online6",online6);
        symbol_table.add_constants();

        expression_t expression;
        expression.register_symbol_table(symbol_table);

        parser_t parser;
        parser.compile(expression_string,expression);

        if (expression.value() > 0)
        {
            time_t currentStamp = (long long)time(/*ago*/0);
            if ((currentStamp-Client(ClientID)->m_AlarmLastTime) > Watchdog(ID)->m_aInterval)
            {
                if (!Client(ClientID)->m_Stats.m_Online4 && !Client(ClientID)->m_Stats.m_Online6)
                {
                    //休眠5分钟如果5分钟后状态发生了变更，消息不发出。
                    printf("download\n");
                }
                Client(ClientID)->m_AlarmLastTime = currentStamp;
                CURL *curl;
                CURLcode res;
                curl_global_init(CURL_GLOBAL_ALL);

                curl = curl_easy_init();
                if(curl) {
                    //standard time
                    char standardTime[32]= { 0 };
                    strftime(standardTime, sizeof(standardTime), "%Y-%m-%d %H:%M:%S",localtime(&currentStamp));

                    //url encode, Rules conflict with url special characters，eg：&, del rules, by https://cpp.la, 2023-10-09
                    char encodeBuffer[2048] = { 0 };
                    sprintf(encodeBuffer, "【告警名称】 %s \n\n【告警时间】 %s  \n\n【用户名】 %s \n\n【节点名】 %s \n\n【虚拟化】 %s \n\n【主机名】 %s \n\n【位  置】 %s",
                            Watchdog(ID)->m_aName,
                            standardTime,
                            username.c_str(),
                            name.c_str(),
                            type.c_str(),
                            host.c_str(),
                            location.c_str());
                    char *encodeUrl = curl_easy_escape(curl, encodeBuffer, strlen(encodeBuffer));

                    //standard url
                    char urlBuffer[2048] = { 0 };
                    sprintf(urlBuffer, "%s%s",Watchdog(ID)->m_aCallback, encodeUrl);


                    curl_easy_setopt(curl, CURLOPT_POST, 1L);
                    curl_easy_setopt(curl, CURLOPT_URL, urlBuffer);
                    curl_easy_setopt(curl, CURLOPT_POSTFIELDS,"signature=ServerStatus");
                    curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0);
                    curl_easy_setopt(curl, CURLOPT_SSL_VERIFYHOST, 0);
                    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 3L);
                    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 6L);
                    res = curl_easy_perform(curl);
                    if(res != CURLE_OK)
                        fprintf(stderr, "watchdog failed: %s\n", curl_easy_strerror(res));
                    if(encodeUrl)
                        curl_free(encodeUrl);
                    curl_easy_cleanup(curl);
                }
                curl_global_cleanup();
            }
        }
        ID++;
    }
}

void CMain::JSONUpdateThread(void *pUser)
{
	CJSONUpdateThreadData *m_pJSONUpdateThreadData = (CJSONUpdateThreadData *)pUser;
	CClient *pClients = m_pJSONUpdateThreadData->pClients;
	CConfig *pConfig = m_pJSONUpdateThreadData->pConfig;

	while(gs_Running)
	{
		char aFileBuf[2048*NET_MAX_CLIENTS];
		char *pBuf = aFileBuf;

		str_format(pBuf, sizeof(aFileBuf), "{\n\"servers\": [\n");
		pBuf += strlen(pBuf);

		for(int i = 0; i < NET_MAX_CLIENTS; i++)
		{
			if(!pClients[i].m_Active || pClients[i].m_Disabled)
				continue;

			if(pClients[i].m_Connected)
			{
				// Uptime
				char aUptime[16];
				int Days = pClients[i].m_Stats.m_Uptime/60.0/60.0/24.0;
				if(Days > 0)
				{
					if(Days > 1)
						str_format(aUptime, sizeof(aUptime), "%d 天", Days);
					else
						str_format(aUptime, sizeof(aUptime), "%d 天", Days);
				}
				else
					str_format(aUptime, sizeof(aUptime), "%02d:%02d:%02d", (int)(pClients[i].m_Stats.m_Uptime/60.0/60.0), (int)((pClients[i].m_Stats.m_Uptime/60)%60), (int)((pClients[i].m_Stats.m_Uptime)%60));

				// track month network traffic, diff: 2021-10-01 00:05, 5minutes
				// last_network_in/out is last record flag.
                time_t currentStamp = (long long)time(/*ago*/0);
                if(0 == pClients[i].m_LastNetworkIN || (0 != pClients[i].m_Stats.m_NetworkIN && pClients[i].m_LastNetworkIN > pClients[i].m_Stats.m_NetworkIN) || (localtime(&currentStamp)->tm_mday == pClients[i].m_aMonthStart && localtime(&currentStamp)->tm_hour == 0 && localtime(&currentStamp)->tm_min < 5))
                {
                    pClients[i].m_LastNetworkIN = pClients[i].m_Stats.m_NetworkIN;
                    pClients[i].m_LastNetworkOUT = pClients[i].m_Stats.m_NetworkOUT;
                }

				str_format(pBuf, sizeof(aFileBuf) - (pBuf - aFileBuf),
                 "{ \"name\": \"%s\",\"type\": \"%s\",\"host\": \"%s\",\"location\": \"%s\",\"online4\": %s, \"online6\": %s, \"uptime\": \"%s\",\"load_1\": %.2f, \"load_5\": %.2f, \"load_15\": %.2f,\"ping_10010\": %.2f, \"ping_189\": %.2f, \"ping_10086\": %.2f,\"time_10010\": %" PRId64 ", \"time_189\": %" PRId64 ", \"time_10086\": %" PRId64 ", \"tcp_count\": %" PRId64 ", \"udp_count\": %" PRId64 ", \"process_count\": %" PRId64 ", \"thread_count\": %" PRId64 ", \"network_rx\": %" PRId64 ", \"network_tx\": %" PRId64 ", \"network_in\": %" PRId64 ", \"network_out\": %" PRId64 ", \"cpu\": %d, \"memory_total\": %" PRId64 ", \"memory_used\": %" PRId64 ", \"swap_total\": %" PRId64 ", \"swap_used\": %" PRId64 ", \"hdd_total\": %" PRId64 ", \"hdd_used\": %" PRId64 ", \"last_network_in\": %" PRId64 ", \"last_network_out\": %" PRId64 ",\"io_read\": %" PRId64 ", \"io_write\": %" PRId64 ",\"custom\": \"%s\", \"os\": \"%s\" },\n",
					pClients[i].m_aName,pClients[i].m_aType,pClients[i].m_aHost,pClients[i].m_aLocation,
					pClients[i].m_Stats.m_Online4 ? "true" : "false",pClients[i].m_Stats.m_Online6 ? "true" : "false",
					aUptime, pClients[i].m_Stats.m_Load_1, pClients[i].m_Stats.m_Load_5, pClients[i].m_Stats.m_Load_15, pClients[i].m_Stats.m_ping_10010, pClients[i].m_Stats.m_ping_189, pClients[i].m_Stats.m_ping_10086,
					pClients[i].m_Stats.m_time_10010, pClients[i].m_Stats.m_time_189, pClients[i].m_Stats.m_time_10086,pClients[i].m_Stats.m_tcpCount,pClients[i].m_Stats.m_udpCount,pClients[i].m_Stats.m_processCount,pClients[i].m_Stats.m_threadCount,
					pClients[i].m_Stats.m_NetworkRx, pClients[i].m_Stats.m_NetworkTx, pClients[i].m_Stats.m_NetworkIN, pClients[i].m_Stats.m_NetworkOUT, (int)pClients[i].m_Stats.m_CPU, pClients[i].m_Stats.m_MemTotal, pClients[i].m_Stats.m_MemUsed,
					pClients[i].m_Stats.m_SwapTotal, pClients[i].m_Stats.m_SwapUsed, pClients[i].m_Stats.m_HDDTotal, pClients[i].m_Stats.m_HDDUsed,
					pClients[i].m_Stats.m_NetworkIN == 0 || pClients[i].m_LastNetworkIN == 0 ? pClients[i].m_Stats.m_NetworkIN : pClients[i].m_LastNetworkIN,
					pClients[i].m_Stats.m_NetworkOUT == 0 || pClients[i].m_LastNetworkOUT == 0 ? pClients[i].m_Stats.m_NetworkOUT : pClients[i].m_LastNetworkOUT,
					pClients[i].m_Stats.m_IORead, pClients[i].m_Stats.m_IOWrite,
					pClients[i].m_Stats.m_aCustom,
					pClients[i].m_Stats.m_aOS[0] ? pClients[i].m_Stats.m_aOS : "");
				pBuf += strlen(pBuf);
			}
			else
			{
			    // sava network traffic record to json when close client
			    // last_network_in == last network in record, last_network_out == last network out record
				str_format(pBuf, sizeof(aFileBuf) - (pBuf - aFileBuf), "{ \"name\": \"%s\", \"type\": \"%s\", \"host\": \"%s\", \"location\": \"%s\", \"online4\": false, \"online6\": false, \"last_network_in\": %" PRId64 ", \"last_network_out\": %" PRId64 ", \"os\": \"%s\" },\n",
					pClients[i].m_aName, pClients[i].m_aType, pClients[i].m_aHost, pClients[i].m_aLocation, pClients[i].m_LastNetworkIN, pClients[i].m_LastNetworkOUT,
					pClients[i].m_Stats.m_aOS[0] ? pClients[i].m_Stats.m_aOS : "");
				pBuf += strlen(pBuf);
			}
		}
		// append ssl certs data
		str_format(pBuf - 2, sizeof(aFileBuf) - (pBuf - aFileBuf), "\n],\n\"sslcerts\": [\n");
		pBuf += strlen(pBuf);
		for(int si = 0; si < NET_MAX_CLIENTS; si++)
		{
			if(!m_pJSONUpdateThreadData->pMain->SSLCert(si) || !strcmp(m_pJSONUpdateThreadData->pMain->SSLCert(si)->m_aName, "NULL")) break;
			int64_t expire_ts = m_pJSONUpdateThreadData->pMain->SSLCert(si)->m_aExpireTS;
			int expire_days = 0;
			if(expire_ts>0){
				int64_t nowts = (long long)time(/*ago*/0);
				expire_days = (int)((expire_ts - nowts)/86400);
			}
			str_format(pBuf, sizeof(aFileBuf) - (pBuf - aFileBuf), "{ \"name\": \"%s\", \"domain\": \"%s\", \"port\": %d, \"expire_ts\": %lld, \"expire_days\": %d, \"mismatch\": %s },\n", m_pJSONUpdateThreadData->pMain->SSLCert(si)->m_aName, m_pJSONUpdateThreadData->pMain->SSLCert(si)->m_aDomain, m_pJSONUpdateThreadData->pMain->SSLCert(si)->m_aPort, (long long)expire_ts, expire_days, m_pJSONUpdateThreadData->pMain->SSLCert(si)->m_aHostnameMismatch?"true":"false");
			pBuf += strlen(pBuf);
		}
		if(pBuf - aFileBuf >= 2) str_format(pBuf - 2, sizeof(aFileBuf) - (pBuf - aFileBuf), "\n],\n\"updated\": \"%lld\"%s\n}", (long long)time(/*ago*/0), m_pJSONUpdateThreadData->m_ReloadRequired?",\n\"reload\": true":"");
		if(m_pJSONUpdateThreadData->m_ReloadRequired) m_pJSONUpdateThreadData->m_ReloadRequired--;
		pBuf += strlen(pBuf);

		char aJSONFileTmp[1024];
		str_format(aJSONFileTmp, sizeof(aJSONFileTmp), "%s~", pConfig->m_aJSONFile);
		IOHANDLE File = io_open(aJSONFileTmp, IOFLAG_WRITE);
		if(!File)
		{
			dbg_msg("main", "Couldn't open %s", aJSONFileTmp);
			exit(1);
		}
		io_write(File, aFileBuf, (pBuf - aFileBuf));
		io_flush(File);
		io_close(File);
		fs_rename(aJSONFileTmp, pConfig->m_aJSONFile);
		thread_sleep(1000);
	}
	// support by: https://cpp.la. don't remove month traffic record, storage as "stats.json~", remark: 2021-10-18
	// fs_remove(pConfig->m_aJSONFile);
    char aJSONFileTmp[1024];
    str_format(aJSONFileTmp, sizeof(aJSONFileTmp), "%s~", pConfig->m_aJSONFile);
    fs_rename(pConfig->m_aJSONFile, aJSONFileTmp);
}

void CMain::offlineAlarmThread(void *pUser)
{
    CJSONUpdateThreadData *m_OfflineAlarmThreadData = (CJSONUpdateThreadData *)pUser;
    CClient *pClients = m_OfflineAlarmThreadData->pClients;
    CWatchDog *pWatchDogs = m_OfflineAlarmThreadData->pWatchDogs;
    volatile short ClientID = m_OfflineAlarmThreadData->m_ReloadRequired;
    thread_sleep(25000);
    if(!pClients[ClientID].m_Connected)
    {
        int ID = 0;
        while (strcmp(pWatchDogs[ID].m_aName, "NULL"))
        {
            typedef exprtk::symbol_table<double> symbol_table_t;
            typedef exprtk::expression<double>   expression_t;
            typedef exprtk::parser<double>       parser_t;
            const std::string expression_string = pWatchDogs[ID].m_aRule;
            std::string username = pClients[ClientID].m_aUsername;
            std::string name = pClients[ClientID].m_aName;
            std::string type = pClients[ClientID].m_aType;
            std::string host = pClients[ClientID].m_aHost;
            std::string location = pClients[ClientID].m_aLocation;
            std::double_t online4 = pClients[ClientID].m_Stats.m_Online4;
            std::double_t online6 = pClients[ClientID].m_Stats.m_Online6;

            symbol_table_t symbol_table;
            symbol_table.add_stringvar("username", username);
            symbol_table.add_stringvar("name", name);
            symbol_table.add_stringvar("type", type);
            symbol_table.add_stringvar("host", host);
            symbol_table.add_stringvar("location", location);
            symbol_table.add_variable("online4",online4);
            symbol_table.add_variable("online6",online6);
            symbol_table.add_constants();

            expression_t expression;
            expression.register_symbol_table(symbol_table);

            parser_t parser;
            parser.compile(expression_string,expression);

            if (expression.value() > 0)
            {
                time_t currentStamp = (long long)time(/*ago*/0);
                if ((currentStamp-pClients[ClientID].m_AlarmLastTime) > pWatchDogs[ID].m_aInterval)
                {
                    printf("客户端下线且超过阈值, Client disconnects and sends alert information\n");
                    pClients[ClientID].m_AlarmLastTime = currentStamp;
                    CURL *curl;
                    CURLcode res;
                    curl_global_init(CURL_GLOBAL_ALL);

                    curl = curl_easy_init();
                    if(curl) {
                        //standard time
                        char standardTime[32]= { 0 };
                        strftime(standardTime, sizeof(standardTime), "%Y-%m-%d %H:%M:%S",localtime(&currentStamp));

                        //url encode, Rules conflict with url special characters，eg：&, del rules, by https://cpp.la, 2023-10-09
                        char encodeBuffer[2048] = { 0 };
                        sprintf(encodeBuffer, "【告警名称】 %s \n\n【告警时间】 %s  \n\n【用户名】 %s \n\n【节点名】 %s \n\n【虚拟化】 %s \n\n【主机名】 %s \n\n【位  置】 %s",
                                pWatchDogs[ID].m_aName,
                                standardTime,
                                pClients[ClientID].m_aUsername,
                                pClients[ClientID].m_aName,
                                pClients[ClientID].m_aType,
                                pClients[ClientID].m_aHost,
                                pClients[ClientID].m_aLocation);
                        char *encodeUrl = curl_easy_escape(curl, encodeBuffer, strlen(encodeBuffer));

                        //standard url
                        char urlBuffer[2048] = { 0 };
                        sprintf(urlBuffer, "%s%s",pWatchDogs[ID].m_aCallback, encodeUrl);


                        curl_easy_setopt(curl, CURLOPT_POST, 1L);
                        curl_easy_setopt(curl, CURLOPT_URL, urlBuffer);
                        curl_easy_setopt(curl, CURLOPT_POSTFIELDS,"signature=ServerStatus");
                        curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0);
                        curl_easy_setopt(curl, CURLOPT_SSL_VERIFYHOST, 0);
                        curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 3L);
                        curl_easy_setopt(curl, CURLOPT_TIMEOUT, 6L);
                        res = curl_easy_perform(curl);
                        if(res != CURLE_OK)
                            fprintf(stderr, "watchdog failed: %s\n", curl_easy_strerror(res));
                        if(encodeUrl)
                            curl_free(encodeUrl);
                        curl_easy_cleanup(curl);
                    }
                    curl_global_cleanup();
                }
                else
                    printf("客户端下线但未超过阈值，No alarm if the threshold is not exceeded\n");
            }
            ID++;
        }
    }
    else
    {
        printf("网络波动，No alarm information is sent due to network fluctuations\n");
    }
    fflush(stdout);
}

int CMain::ReadConfig()
{
	// read and parse config
	IOHANDLE File = io_open(m_Config.m_aConfigFile, IOFLAG_READ);
	if(!File)
	{
		dbg_msg("main", "Couldn't open %s", m_Config.m_aConfigFile);
		return 1;
	}
	int FileSize = (int)io_length(File);
	char *pFileData = (char *)mem_alloc(FileSize + 1, 1);

	io_read(File, pFileData, FileSize);
	pFileData[FileSize] = 0;
	io_close(File);

	// parse json data
	json_settings JsonSettings;
	mem_zero(&JsonSettings, sizeof(JsonSettings));
	char aError[256];
	json_value *pJsonData = json_parse_ex(&JsonSettings, pFileData, strlen(pFileData), aError);
	if(!pJsonData)
	{
		dbg_msg("main", "JSON Error in file %s: %s", m_Config.m_aConfigFile, aError);
		mem_free(pFileData);
		return 1;
	}

	// reset clients
	for(int i = 0; i < NET_MAX_CLIENTS; i++)
	{
		if(!Client(i)->m_Active || !Client(i)->m_Connected)
			continue;

		m_Server.Network()->Drop(Client(i)->m_ClientNetID, "Server reloading...");
	}
	mem_zero(m_aClients, sizeof(m_aClients));
	for(int i = 0; i < NET_MAX_CLIENTS; i++)
		m_aClients[i].m_ClientNetID = -1;

	// extract data
	int ID = 0;
	const json_value &rStart = (*pJsonData)["servers"];
	if(rStart.type == json_array)
	{
		for(unsigned i = 0; i < rStart.u.array.length; i++)
		{
			if(ID < 0 || ID >= NET_MAX_CLIENTS)
				continue;

			Client(ID)->m_Active = true;
			Client(ID)->m_Disabled = rStart[i]["disabled"].u.boolean;
			str_copy(Client(ID)->m_aName, rStart[i]["name"].u.string.ptr, sizeof(Client(ID)->m_aName));
			str_copy(Client(ID)->m_aUsername, rStart[i]["username"].u.string.ptr, sizeof(Client(ID)->m_aUsername));
			str_copy(Client(ID)->m_aType, rStart[i]["type"].u.string.ptr, sizeof(Client(ID)->m_aType));
			str_copy(Client(ID)->m_aHost, rStart[i]["host"].u.string.ptr, sizeof(Client(ID)->m_aHost));
			str_copy(Client(ID)->m_aLocation, rStart[i]["location"].u.string.ptr, sizeof(Client(ID)->m_aLocation));
			str_copy(Client(ID)->m_aPassword, rStart[i]["password"].u.string.ptr, sizeof(Client(ID)->m_aPassword));
			//if month start day > 28, diff: 3days(29,30,31)
            Client(ID)->m_aMonthStart = rStart[i]["monthstart"].u.integer;
            if(Client(ID)->m_aMonthStart > 28)
            {
                Client(ID)->m_aMonthStart = 28;
            }
            Client(ID)->m_LastNetworkIN = 0;
            Client(ID)->m_LastNetworkOUT = 0;

			if(m_Config.m_Verbose)
			{
				if(Client(ID)->m_Disabled)
					dbg_msg("main", "[#%d: Name: \"%s\", Username: \"%s\", Type: \"%s\", Host: \"%s\", Location: \"%s\", Password: \"%s\", MonthStart: %\" PRId64 \"]",
						ID, Client(ID)->m_aName, Client(ID)->m_aUsername, Client(ID)->m_aType, Client(ID)->m_aHost, Client(ID)->m_aLocation, Client(ID)->m_aPassword, Client(ID)->m_aMonthStart);
				else
					dbg_msg("main", "#%d: Name: \"%s\", Username: \"%s\", Type: \"%s\", Host: \"%s\", Location: \"%s\", Password: \"%s\", MonthStart: %\" PRId64 \"",
						ID, Client(ID)->m_aName, Client(ID)->m_aUsername, Client(ID)->m_aType, Client(ID)->m_aHost, Client(ID)->m_aLocation, Client(ID)->m_aPassword, Client(ID)->m_aMonthStart);

			}
			ID++;
		}
	}

	// watch dog
	// support by: https://cpp.la
	ID = 0;
	const json_value &jStart = (*pJsonData)["watchdog"];
	if(jStart.type == json_array)
    {
        for(unsigned i = 0; i < jStart.u.array.length; i++)
        {
            if(ID < 0 || ID >= NET_MAX_CLIENTS)
                continue;

            str_copy(Watchdog(ID)->m_aName, jStart[i]["name"].u.string.ptr, sizeof(Watchdog(ID)->m_aName));
            str_copy(Watchdog(ID)->m_aRule, jStart[i]["rule"].u.string.ptr, sizeof(Watchdog(ID)->m_aRule));
            Watchdog(ID)->m_aInterval = jStart[i]["interval"].u.integer;
            str_copy(Watchdog(ID)->m_aCallback, jStart[i]["callback"].u.string.ptr, sizeof(Watchdog(ID)->m_aCallback));

            ID++;
        }
        str_copy(Watchdog(ID)->m_aName, "NULL", sizeof(Watchdog(ID)->m_aName));
	} 
	else
	{
		str_copy(Watchdog(ID)->m_aName, "NULL", sizeof(Watchdog(ID)->m_aName));
	}

    // monitor
    // support by: https://cpp.la
    ID = 0;
    const json_value &mStart = (*pJsonData)["monitors"];
    if(mStart.type == json_array)
    {
        for(unsigned i = 0; i < mStart.u.array.length; i++)
        {
            if(ID < 0 || ID >= NET_MAX_CLIENTS)
                continue;

            str_copy(Monitors(ID)->m_aName, mStart[i]["name"].u.string.ptr, sizeof(Monitors(ID)->m_aName));
            str_copy(Monitors(ID)->m_aHost, mStart[i]["host"].u.string.ptr, sizeof(Monitors(ID)->m_aHost));
            Monitors(ID)->m_aInterval = mStart[i]["interval"].u.integer;
            str_copy(Monitors(ID)->m_aType, mStart[i]["type"].u.string.ptr, sizeof(Monitors(ID)->m_aType));

            ID++;
        }
        str_copy(Monitors(ID)->m_aName, "NULL", sizeof(Monitors(ID)->m_aName));
	} 
	else
	{
		str_copy(Monitors(ID)->m_aName, "NULL", sizeof(Monitors(ID)->m_aName));
	}

	// sslcerts
	ID = 0;
	const json_value &sStart = (*pJsonData)["sslcerts"];
	if(sStart.type == json_array)
	{
		for(unsigned i = 0; i < sStart.u.array.length; i++)
		{
			if(ID < 0 || ID >= NET_MAX_CLIENTS)
				continue;
			str_copy(SSLCert(ID)->m_aName, sStart[i]["name"].u.string.ptr, sizeof(SSLCert(ID)->m_aName));
			str_copy(SSLCert(ID)->m_aDomain, sStart[i]["domain"].u.string.ptr, sizeof(SSLCert(ID)->m_aDomain));
			SSLCert(ID)->m_aPort = sStart[i]["port"].u.integer;
			SSLCert(ID)->m_aInterval = sStart[i]["interval"].u.integer;
			str_copy(SSLCert(ID)->m_aCallback, sStart[i]["callback"].u.string.ptr, sizeof(SSLCert(ID)->m_aCallback));
			SSLCert(ID)->m_aExpireTS = 0; // reset
			SSLCert(ID)->m_aLastCheck = 0;
			SSLCert(ID)->m_aLastAlarm7 = 0;
			SSLCert(ID)->m_aLastAlarm3 = 0;
			SSLCert(ID)->m_aLastAlarm1 = 0;
			ID++;
		}
		str_copy(SSLCert(ID)->m_aName, "NULL", sizeof(SSLCert(ID)->m_aName));
	}else
		str_copy(SSLCert(ID)->m_aName, "NULL", sizeof(SSLCert(ID)->m_aName));

	// if file exists, read last network traffic record，reset m_LastNetworkIN and m_LastNetworkOUT
	// support by: https://cpp.la
    IOHANDLE nFile = io_open(m_Config.m_aJSONFile, IOFLAG_READ);
	if(!nFile)
    {
	    char aJSONFileTmp[1024];
        str_format(aJSONFileTmp, sizeof(aJSONFileTmp), "%s~", m_Config.m_aJSONFile);
        nFile = io_open(aJSONFileTmp, IOFLAG_READ);
    }
    if(nFile)
    {
        int nFileSize = (int)io_length(nFile);
        char *pNFileData = (char *)mem_alloc(nFileSize + 1, 1);

        io_read(nFile, pNFileData, nFileSize);
        pNFileData[nFileSize] = 0;
        io_close(nFile);

        json_settings nJsonSettings;
        mem_zero(&nJsonSettings, sizeof(nJsonSettings));
        json_value *pNJsonData = json_parse_ex(&nJsonSettings, pNFileData, strlen(pNFileData), aError);
        if(pNJsonData)
        {
            const json_value &cStart = (*pNJsonData)["servers"];
            if(rStart.type == json_array)
            {
                int ID = 0;
                for(unsigned i = 0; i < rStart.u.array.length; i++)
                {
                    if(ID < 0 || ID >= NET_MAX_CLIENTS)
                        continue;
                    for(unsigned j = 0; j < cStart.u.array.length; j++)
                    {
                        if(strcmp(Client(ID)->m_aName, cStart[j]["name"].u.string.ptr)==0 &&
                            strcmp(Client(ID)->m_aType, cStart[j]["type"].u.string.ptr)==0 &&
                            strcmp(Client(ID)->m_aHost, cStart[j]["host"].u.string.ptr)==0 &&
                            strcmp(Client(ID)->m_aLocation, cStart[j]["location"].u.string.ptr)==0)
                        {
                            Client(ID)->m_LastNetworkIN = cStart[j]["last_network_in"].u.integer;
                            Client(ID)->m_LastNetworkOUT = cStart[j]["last_network_out"].u.integer;
                            break;
                        }
                    }
                    ID++;
                }
            }
            json_value_free(pNJsonData);
        }
        mem_free(pNFileData);
    }

	// clean up
	json_value_free(pJsonData);
	mem_free(pFileData);

	// tell clients to reload the page
	m_JSONUpdateThreadData.m_ReloadRequired = 2;

	return 0;
}

int CMain::Run()
{
	if(m_Server.Init(this, m_Config.m_aBindAddr, m_Config.m_Port))
		return 1;

	if(ReadConfig())
		return 1;

	// Start JSON Update Thread
	m_JSONUpdateThreadData.m_ReloadRequired = 2;
	m_JSONUpdateThreadData.pClients = m_aClients;
	m_JSONUpdateThreadData.pConfig = &m_Config;
    m_JSONUpdateThreadData.pWatchDogs = m_aCWatchDogs;
	m_JSONUpdateThreadData.pMain = this;
	void *LoadThread = thread_create(JSONUpdateThread, &m_JSONUpdateThreadData);
	// Start SSL check thread
	static SSLCheckThreadData sslData; sslData.pMain = this; thread_create(SSLCheckThread, &sslData);
	//thread_detach(LoadThread);

	while(gs_Running)
	{
		if(gs_ReloadConfig)
		{
			if(ReadConfig())
				return 1;
			m_Server.NetBan()->UnbanAll();
			gs_ReloadConfig = 0;
		}

		m_Server.Update();

		// wait for incomming data
		net_socket_read_wait(*m_Server.Network()->Socket(), 10);
	}

	dbg_msg("server", "Closing.");
	m_Server.Network()->Close();
	thread_wait(LoadThread);

	return 0;
}

int main(int argc, const char *argv[])
{
	int RetVal;
	dbg_logger_stdout();

	#if defined(CONF_FAMILY_UNIX)
		signal(SIGINT, ExitFunc);
		signal(SIGTERM, ExitFunc);
		signal(SIGQUIT, ExitFunc);
		signal(SIGHUP, ReloadFunc);
	#endif

	char aUsage[128];
	CConfig Config;
	str_format(aUsage, sizeof(aUsage), "%s [options]", argv[0]);
	const char *pConfigFile = 0;
	const char *pWebDir = 0;
	const char *pBindAddr = 0;

	struct argparse_option aOptions[] = {
		OPT_HELP(),
		OPT_BOOLEAN('v', "verbose", &Config.m_Verbose, "Verbose output", 0),
		OPT_STRING('c', "config", &pConfigFile, "Config file to use", 0),
		OPT_STRING('d', "web-dir", &pWebDir, "Location of the web directory", 0),
		OPT_STRING('b', "bind", &pBindAddr, "Bind to address", 0),
		OPT_INTEGER('p', "port", &Config.m_Port, "Listen on port", 0),
		OPT_END(),
	};
	struct argparse Argparse;
	argparse_init(&Argparse, aOptions, aUsage, 0);
	argc = argparse_parse(&Argparse, argc, argv);

	if(pConfigFile)
		str_copy(Config.m_aConfigFile, pConfigFile, sizeof(Config.m_aConfigFile));
	if(pWebDir)
		str_copy(Config.m_aWebDir, pWebDir, sizeof(Config.m_aWebDir));
	if(pBindAddr)
		str_copy(Config.m_aBindAddr, pBindAddr, sizeof(Config.m_aBindAddr));

	if(Config.m_aWebDir[strlen(Config.m_aWebDir)-1] != '/')
		str_append(Config.m_aWebDir, "/", sizeof(Config.m_aWebDir));
	if(!fs_is_dir(Config.m_aWebDir))
	{
		dbg_msg("main", "ERROR: Can't find web directory: %s", Config.m_aWebDir);
		return 1;
	}

	char aTmp[1024];
	str_format(aTmp, sizeof(aTmp), "%s%s", Config.m_aWebDir, Config.m_aJSONFile);
	str_copy(Config.m_aJSONFile, aTmp, sizeof(Config.m_aJSONFile));

	CMain Main(Config);
	RetVal = Main.Run();

	return RetVal;
}

