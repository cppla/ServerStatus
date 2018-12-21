#define __STDC_FORMAT_MACROS
#include <inttypes.h>
#include <time.h>
#include <detect.h>
#include <system.h>
#include <argparse.h>
#include <json.h>
#include "server.h"
#include "main.h"

#if defined(CONF_FAMILY_UNIX)
	#include <signal.h>
#endif

#ifndef PRId64
	#define PRId64 "I64d"
#endif

static volatile int gs_Running = 1;
static volatile int gs_ReloadConfig = 0;

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
		if(rStart["cpu"].type)
			pClient->m_Stats.m_CPU = rStart["cpu"].u.dbl;
		if(rStart["online4"].type && pClient->m_ClientNetType == NETTYPE_IPV6)
			pClient->m_Stats.m_Online4 = rStart["online4"].u.boolean;
		if(rStart["online6"].type && pClient->m_ClientNetType == NETTYPE_IPV4)
			pClient->m_Stats.m_Online6 = rStart["online6"].u.boolean;
		if(rStart["ip_status"].type)
			pClient->m_Stats.m_IpStatus = rStart["ip_status"].u.boolean;
		if(rStart["custom"].type == json_string)
			str_copy(pClient->m_Stats.m_aCustom, rStart["custom"].u.string.ptr, sizeof(pClient->m_Stats.m_aCustom));

		if(m_Config.m_Verbose)
		{
			if(rStart["online4"].type)
				dbg_msg("main", "Online4: %s\nUptime: %" PRId64 "\nIpStatus: %s\nLoad_1: %f\nLoad_5: %f\nLoad_15: %f\nPing_10010: %f\nPing_189: %f\nPing_10086: %f\nTime_10010: %" PRId64 "\nTime_189: %" PRId64 "\nTime_10086: %" PRId64 "\nTcp_count: %" PRId64 "\nUdp_count: %" PRId64 "\nprocess_count: %" PRId64 "\nthread_count: %" PRId64 "\nNetworkRx: %" PRId64 "\nNetworkTx: %" PRId64 "\nNetworkIN: %" PRId64 "\nNetworkOUT: %" PRId64 "\nMemTotal: %" PRId64 "\nMemUsed: %" PRId64 "\nSwapTotal: %" PRId64 "\nSwapUsed: %" PRId64 "\nHDDTotal: %" PRId64 "\nHDDUsed: %" PRId64 "\nCPU: %f\n",
					rStart["online4"].u.boolean ? "true" : "false",
					pClient->m_Stats.m_Uptime,
					pClient->m_Stats.m_IpStatus ? "true" : "false",
					pClient->m_Stats.m_Load_1, pClient->m_Stats.m_Load_5, pClient->m_Stats.m_Load_15, pClient->m_Stats.m_ping_10010, pClient->m_Stats.m_ping_189, pClient->m_Stats.m_ping_10086, pClient->m_Stats.m_time_10010, pClient->m_Stats.m_time_189, pClient->m_Stats.m_time_10086,pClient->m_Stats.m_tcpCount,pClient->m_Stats.m_udpCount,pClient->m_Stats.m_processCount,pClient->m_Stats.m_threadCount,pClient->m_Stats.m_NetworkRx, pClient->m_Stats.m_NetworkTx, pClient->m_Stats.m_NetworkIN, pClient->m_Stats.m_NetworkOUT, pClient->m_Stats.m_MemTotal, pClient->m_Stats.m_MemUsed, pClient->m_Stats.m_SwapTotal, pClient->m_Stats.m_SwapUsed, pClient->m_Stats.m_HDDTotal, pClient->m_Stats.m_HDDUsed, pClient->m_Stats.m_CPU);
			else if(rStart["online6"].type)
				dbg_msg("main", "Online6: %s\nUptime: %" PRId64 "\nIpStatus: %s\nLoad_1: %f\nLoad_5: %f\nLoad_15: %f\nPing_10010: %f\nPing_189: %f\nPing_10086: %f\nTime_10010: %" PRId64 "\nTime_189: %" PRId64 "\nTime_10086: %" PRId64 "\nTcp_count: %" PRId64 "\nUdp_count: %" PRId64 "\nprocess_count: %" PRId64 "\nthread_count: %" PRId64 "\nNetworkRx: %" PRId64 "\nNetworkTx: %" PRId64 "\nNetworkIN: %" PRId64 "\nNetworkOUT: %" PRId64 "\nMemTotal: %" PRId64 "\nMemUsed: %" PRId64 "\nSwapTotal: %" PRId64 "\nSwapUsed: %" PRId64 "\nHDDTotal: %" PRId64 "\nHDDUsed: %" PRId64 "\nCPU: %f\n",
					rStart["online6"].u.boolean ? "true" : "false",
					pClient->m_Stats.m_Uptime,
					pClient->m_Stats.m_IpStatus ? "true" : "false",
					pClient->m_Stats.m_Load_1, pClient->m_Stats.m_Load_5, pClient->m_Stats.m_Load_15, pClient->m_Stats.m_ping_10010, pClient->m_Stats.m_ping_189, pClient->m_Stats.m_ping_10086, pClient->m_Stats.m_time_10010, pClient->m_Stats.m_time_189, pClient->m_Stats.m_time_10086,pClient->m_Stats.m_tcpCount,pClient->m_Stats.m_udpCount,pClient->m_Stats.m_processCount,pClient->m_Stats.m_threadCount,pClient->m_Stats.m_NetworkRx, pClient->m_Stats.m_NetworkTx, pClient->m_Stats.m_NetworkIN, pClient->m_Stats.m_NetworkOUT, pClient->m_Stats.m_MemTotal, pClient->m_Stats.m_MemUsed, pClient->m_Stats.m_SwapTotal, pClient->m_Stats.m_SwapUsed, pClient->m_Stats.m_HDDTotal, pClient->m_Stats.m_HDDUsed, pClient->m_Stats.m_CPU);
			else
				dbg_msg("main", "Uptime: %" PRId64 "\nIpStatus: %s\nLoad_1: %f\nLoad_5: %f\nLoad_15: %f\nPing_10010: %f\nPing_189: %f\nPing_10086: %f\nTime_10010: %" PRId64 "\nTime_189: %" PRId64 "\nTime_10086: %" PRId64 "\nTcp_count: %" PRId64 "\nUdp_count: %" PRId64 "\nprocess_count: %" PRId64 "\nthread_count: %" PRId64 "\nNetworkRx: %" PRId64 "\nNetworkTx: %" PRId64 "\nNetworkIN: %" PRId64 "\nNetworkOUT: %" PRId64 "\nMemTotal: %" PRId64 "\nMemUsed: %" PRId64 "\nSwapTotal: %" PRId64 "\nSwapUsed: %" PRId64 "\nHDDTotal: %" PRId64 "\nHDDUsed: %" PRId64 "\nCPU: %f\n",
					pClient->m_Stats.m_Uptime,
					pClient->m_Stats.m_IpStatus ? "true" : "false",
					pClient->m_Stats.m_Load_1, pClient->m_Stats.m_Load_5, pClient->m_Stats.m_Load_15, pClient->m_Stats.m_ping_10010, pClient->m_Stats.m_ping_189, pClient->m_Stats.m_ping_10086, pClient->m_Stats.m_time_10010, pClient->m_Stats.m_time_189, pClient->m_Stats.m_time_10086,pClient->m_Stats.m_tcpCount,pClient->m_Stats.m_udpCount,pClient->m_Stats.m_processCount,pClient->m_Stats.m_threadCount,pClient->m_Stats.m_NetworkRx, pClient->m_Stats.m_NetworkTx, pClient->m_Stats.m_NetworkIN, pClient->m_Stats.m_NetworkOUT, pClient->m_Stats.m_MemTotal, pClient->m_Stats.m_MemUsed, pClient->m_Stats.m_SwapTotal, pClient->m_Stats.m_SwapUsed, pClient->m_Stats.m_HDDTotal, pClient->m_Stats.m_HDDUsed, pClient->m_Stats.m_CPU);
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

				str_format(pBuf, sizeof(aFileBuf) - (pBuf - aFileBuf),
				 "{ \"name\": \"%s\",\"type\": \"%s\",\"host\": \"%s\",\"location\": \"%s\",\"online4\": %s, \"online6\": %s,\"ip_status\": %s,\"uptime\": \"%s\",\"load_1\": %.2f, \"load_5\": %.2f, \"load_15\": %.2f,\"ping_10010\": %.2f, \"ping_189\": %.2f, \"ping_10086\": %.2f,\"time_10010\": %" PRId64 ", \"time_189\": %" PRId64 ", \"time_10086\": %" PRId64 ", \"tcp_count\": %" PRId64 ", \"udp_count\": %" PRId64 ", \"process_count\": %" PRId64 ", \"thread_count\": %" PRId64 ", \"network_rx\": %" PRId64 ", \"network_tx\": %" PRId64 ", \"network_in\": %" PRId64 ", \"network_out\": %" PRId64 ", \"cpu\": %d, \"memory_total\": %" PRId64 ", \"memory_used\": %" PRId64 ", \"swap_total\": %" PRId64 ", \"swap_used\": %" PRId64 ", \"hdd_total\": %" PRId64 ", \"hdd_used\": %" PRId64 ", \"custom\": \"%s\" },\n",
					pClients[i].m_aName,pClients[i].m_aType,pClients[i].m_aHost,pClients[i].m_aLocation,
					pClients[i].m_Stats.m_Online4 ? "true" : "false",pClients[i].m_Stats.m_Online6 ? "true" : "false",pClients[i].m_Stats.m_IpStatus ? "true": "false",
					aUptime, pClients[i].m_Stats.m_Load_1, pClients[i].m_Stats.m_Load_5, pClients[i].m_Stats.m_Load_15, pClients[i].m_Stats.m_ping_10010, pClients[i].m_Stats.m_ping_189, pClients[i].m_Stats.m_ping_10086, pClients[i].m_Stats.m_time_10010, pClients[i].m_Stats.m_time_189, pClients[i].m_Stats.m_time_10086,pClients[i].m_Stats.m_tcpCount,pClients[i].m_Stats.m_udpCount,pClients[i].m_Stats.m_processCount,pClients[i].m_Stats.m_threadCount,pClients[i].m_Stats.m_NetworkRx, pClients[i].m_Stats.m_NetworkTx, pClients[i].m_Stats.m_NetworkIN, pClients[i].m_Stats.m_NetworkOUT, (int)pClients[i].m_Stats.m_CPU, pClients[i].m_Stats.m_MemTotal, pClients[i].m_Stats.m_MemUsed, pClients[i].m_Stats.m_SwapTotal, pClients[i].m_Stats.m_SwapUsed, pClients[i].m_Stats.m_HDDTotal, pClients[i].m_Stats.m_HDDUsed, pClients[i].m_Stats.m_aCustom);
				pBuf += strlen(pBuf);
			}
			else
			{
				str_format(pBuf, sizeof(aFileBuf) - (pBuf - aFileBuf), "{ \"name\": \"%s\", \"type\": \"%s\", \"host\": \"%s\", \"location\": \"%s\", \"online4\": false, \"online6\": false },\n",
					pClients[i].m_aName, pClients[i].m_aType, pClients[i].m_aHost, pClients[i].m_aLocation);
				pBuf += strlen(pBuf);
			}
		}
		if(!m_pJSONUpdateThreadData->m_ReloadRequired)
			str_format(pBuf - 2, sizeof(aFileBuf) - (pBuf - aFileBuf), "\n],\n\"updated\": \"%lld\"\n}", (long long)time(/*ago*/0));
		else
		{
			str_format(pBuf - 2, sizeof(aFileBuf) - (pBuf - aFileBuf), "\n],\n\"updated\": \"%lld\",\n\"reload\": true\n}", (long long)time(/*ago*/0));
			m_pJSONUpdateThreadData->m_ReloadRequired--;
		}
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
	fs_remove(pConfig->m_aJSONFile);
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

			if(m_Config.m_Verbose)
			{
				if(Client(ID)->m_Disabled)
					dbg_msg("main", "[#%d: Name: \"%s\", Username: \"%s\", Type: \"%s\", Host: \"%s\", Location: \"%s\", Password: \"%s\"]",
						ID, Client(ID)->m_aName, Client(ID)->m_aUsername, Client(ID)->m_aType, Client(ID)->m_aHost, Client(ID)->m_aLocation, Client(ID)->m_aPassword);
				else
					dbg_msg("main", "#%d: Name: \"%s\", Username: \"%s\", Type: \"%s\", Host: \"%s\", Location: \"%s\", Password: \"%s\"",
						ID, Client(ID)->m_aName, Client(ID)->m_aUsername, Client(ID)->m_aType, Client(ID)->m_aHost, Client(ID)->m_aLocation, Client(ID)->m_aPassword);

			}
			ID++;
		}
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
	void *LoadThread = thread_create(JSONUpdateThread, &m_JSONUpdateThreadData);
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
