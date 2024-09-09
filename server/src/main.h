#ifndef MAIN_H
#define MAIN_H

#include <stdint.h>
#include "server.h"

class CConfig
{
public:
	bool m_Verbose;
	char m_aConfigFile[1024];
	char m_aWebDir[1024];
	char m_aTemplateFile[1024];
	char m_aJSONFile[1024];
	char m_aBindAddr[256];
	int m_Port;

	CConfig();
};

class CMain
{
	CConfig m_Config;
	CServer m_Server;

	struct CClient
	{
		bool m_Active;
		bool m_Disabled;
		bool m_Connected;
		int m_ClientNetID;
		int m_ClientNetType;
		char m_aUsername[128];
		char m_aName[128];
		char m_aType[128];
		char m_aHost[128];
		char m_aLocation[128];
		char m_aPassword[128];
        int m_aMonthStart;          //track month network traffic. by: https://cpp.la

        int64_t m_LastNetworkIN;    //restore month traffic info.
        int64_t m_LastNetworkOUT;   //restore month traffic info.
		int64_t m_TimeConnected;
		int64_t m_LastUpdate;
        int64_t m_AlarmLastTime;    //record last alarm time.

		struct CStats
		{
			bool m_Online4;
			bool m_Online6;
			// bool m_IpStatus， delete ip_status check, Duplicate packet loss rate detection
			// mh361 or mh370, mourn mh370, 2014-03-08 01:20　lost from all over the world. by:https://cpp.la
			int64_t m_Uptime;
			double m_Load_1;
			double m_Load_5;
			double m_Load_15;
			double m_ping_10010;
			double m_ping_189;
			double m_ping_10086;
			int64_t m_time_10010;
			int64_t m_time_189;
			int64_t m_time_10086;
			int64_t m_NetworkRx;
			int64_t m_NetworkTx;
			int64_t m_NetworkIN;
			int64_t m_NetworkOUT;
			int64_t m_MemTotal;
			int64_t m_MemUsed;
			int64_t m_SwapTotal;
			int64_t m_SwapUsed;
			int64_t m_HDDTotal;
			int64_t m_HDDUsed;
			int64_t m_tcpCount;
			int64_t m_udpCount;
			int64_t m_processCount;
			int64_t m_threadCount;
			int64_t m_IORead;
			int64_t m_IOWrite;
			double m_CPU;
			char m_aCustom[1024];
			// Options
			bool m_Pong;
		} m_Stats;
	} m_aClients[NET_MAX_CLIENTS];

	struct CWatchDog{
	    char m_aName[128];
	    char m_aRule[128];
        int  m_aInterval;
	    char m_aCallback[1024];
	} m_aCWatchDogs[NET_MAX_CLIENTS];

    struct CMonitors{
        char m_aName[128];
        char m_aHost[128];
        int  m_aInterval;
        char m_aType[128];
    } m_aCMonitors[NET_MAX_CLIENTS];

	struct CJSONUpdateThreadData
	{
		CClient *pClients;
		CConfig *pConfig;
        CWatchDog *pWatchDogs;
		volatile short m_ReloadRequired;
	} m_JSONUpdateThreadData, m_OfflineAlarmThreadData;

	static void JSONUpdateThread(void *pUser);
    static void offlineAlarmThread(void *pUser);
public:
	CMain(CConfig Config);

	void OnNewClient(int ClienNettID, int ClientID);
	void OnDelClient(int ClientNetID);
	int HandleMessage(int ClientNetID, char *pMessage);
	int ReadConfig();
	int Run();

    CWatchDog *Watchdog(int ruleID) { return &m_aCWatchDogs[ruleID]; }
    CMonitors *Monitors(int ruleID) { return &m_aCMonitors[ruleID]; }

    void WatchdogMessage(int ClientNetID,
                         double load_1, double load_5, double load_15, double ping_10010, double ping_189, double ping_10086,
                         double time_10010, double time_189, double time_10086, double tcp_count, double udp_count, double process_count, double thread_count,
                         double network_rx, double network_tx, double network_in, double network_out, double last_network_in, double last_network_out,
                         double memory_total, double memory_used,double swap_total, double swap_used, double hdd_total,
                         double hdd_used, double io_read, double io_write, double cpu,double online4, double online6);

	CClient *Client(int ClientID) { return &m_aClients[ClientID]; }
	CClient *ClientNet(int ClientNetID);
	const CConfig *Config() const { return &m_Config; }
	int ClientNetToClient(int ClientNetID);
};


#endif
