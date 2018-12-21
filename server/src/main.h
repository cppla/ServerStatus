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

		int64 m_TimeConnected;
		int64 m_LastUpdate;

		struct CStats
		{
			bool m_Online4;
			bool m_Online6;
			bool m_IpStatus;    //mh361 or mh370, mourn mh370, 2014-03-08 01:20　lost from all over the world. by:cpp.la
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
			double m_CPU;
			char m_aCustom[512];
			// Options
			bool m_Pong;
		} m_Stats;
	} m_aClients[NET_MAX_CLIENTS];

	struct CJSONUpdateThreadData
	{
		CClient *pClients;
		CConfig *pConfig;
		volatile short m_ReloadRequired;
	} m_JSONUpdateThreadData;

	static void JSONUpdateThread(void *pUser);
public:
	CMain(CConfig Config);

	void OnNewClient(int ClienNettID, int ClientID);
	void OnDelClient(int ClientNetID);
	int HandleMessage(int ClientNetID, char *pMessage);
	int ReadConfig();
	int Run();

	CClient *Client(int ClientID) { return &m_aClients[ClientID]; }
	CClient *ClientNet(int ClientNetID);
	const CConfig *Config() const { return &m_Config; }
	int ClientNetToClient(int ClientNetID);
};


#endif
