#ifndef SERVER_H
#define SERVER_H

#include "netban.h"
#include "network.h"

class CServer
{
	class CClient
	{
	public:
		enum
		{
			STATE_EMPTY=0,
			STATE_CONNECTED,
			STATE_AUTHED,
		};

		int m_State;
		int64 m_TimeConnected;
		int64 m_LastReceived;
	};
	CClient m_aClients[NET_MAX_CLIENTS];

	CNetwork m_Network;
	CNetBan m_NetBan;

	class CMain *m_pMain;

	bool m_Ready;

	static int NewClientCallback(int ClientID, void *pUser);
	static int DelClientCallback(int ClientID, const char *pReason, void *pUser);

public:
	int Init(CMain *pMain, const char *Bind, int Port);
	void Update();
	void Send(int ClientID, const char *pLine);
	void Shutdown();

	CNetwork *Network() { return &m_Network; }
	CNetBan *NetBan() { return &m_NetBan; }
	CMain *Main() { return m_pMain; }
};

#endif
