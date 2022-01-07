#ifndef NETWORK_H
#define NETWORK_H

enum
{
	NET_CONNSTATE_OFFLINE=0,
	NET_CONNSTATE_CONNECT=1,
	NET_CONNSTATE_PENDING=2,
	NET_CONNSTATE_ONLINE=3,
	NET_CONNSTATE_ERROR=4,

	NET_MAX_PACKETSIZE = 1400,
	NET_MAX_CLIENTS = 256
};

typedef int (*NETFUNC_DELCLIENT)(int ClientID, const char* pReason, void *pUser);
typedef int (*NETFUNC_NEWCLIENT)(int ClientID, void *pUser);

class CNetworkClient
{
private:
	int m_State;

	NETADDR m_PeerAddr;
	NETSOCKET m_Socket;

	char m_aBuffer[NET_MAX_PACKETSIZE];
	int m_BufferOffset;

	char m_aErrorString[256];

	bool m_LineEndingDetected;
	char m_aLineEnding[3];

public:
	void Init(NETSOCKET Socket, const NETADDR *pAddr);
	void Disconnect(const char *pReason);

	int State() const { return m_State; }
	const NETADDR *PeerAddress() const { return &m_PeerAddr; }
	const char *ErrorString() const { return m_aErrorString; }

	void Reset();
	int Update();
	int Send(const char *pLine);
	int Recv(char *pLine, int MaxLength);
};

class CNetwork
{
private:
	struct CSlot
	{
		CNetworkClient m_Connection;
	};

	NETSOCKET m_Socket;
	class CNetBan *m_pNetBan;
	CSlot m_aSlots[NET_MAX_CLIENTS];

	NETFUNC_NEWCLIENT m_pfnNewClient;
	NETFUNC_DELCLIENT m_pfnDelClient;
	void *m_UserPtr;

public:
	void SetCallbacks(NETFUNC_NEWCLIENT pfnNewClient, NETFUNC_DELCLIENT pfnDelClient, void *pUser);

	//
	bool Open(NETADDR BindAddr, CNetBan *pNetBan);
	int Close();

	//
	int Recv(char *pLine, int MaxLength, int *pClientID = 0);
	int Send(int ClientID, const char *pLine);
	int Update();

	//
	int AcceptClient(NETSOCKET Socket, const NETADDR *pAddr);
	int Drop(int ClientID, const char *pReason);

	// status requests
	const NETADDR *ClientAddr(int ClientID) const { return m_aSlots[ClientID].m_Connection.PeerAddress(); }
	const NETSOCKET *Socket() const { return &m_Socket; }
	class CNetBan *NetBan() const { return m_pNetBan; }
};

#endif
