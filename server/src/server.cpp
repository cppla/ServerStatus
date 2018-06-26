#include <system.h>
#include "netban.h"
#include "network.h"
#include "main.h"
#include "server.h"

int CServer::NewClientCallback(int ClientID, void *pUser)
{
	CServer *pThis = (CServer *)pUser;

	char aAddrStr[NETADDR_MAXSTRSIZE];
	net_addr_str(pThis->m_Network.ClientAddr(ClientID), aAddrStr, sizeof(aAddrStr), true);
	if(pThis->Main()->Config()->m_Verbose)
		dbg_msg("server", "Connection accepted. ncid=%d addr=%s'", ClientID, aAddrStr);

	pThis->m_aClients[ClientID].m_State = CClient::STATE_CONNECTED;
	pThis->m_aClients[ClientID].m_TimeConnected = time_get();
	pThis->m_Network.Send(ClientID, "Authentication required:");

	return 0;
}

int CServer::DelClientCallback(int ClientID, const char *pReason, void *pUser)
{
	CServer *pThis = (CServer *)pUser;

	char aAddrStr[NETADDR_MAXSTRSIZE];
	net_addr_str(pThis->m_Network.ClientAddr(ClientID), aAddrStr, sizeof(aAddrStr), true);
	if(pThis->Main()->Config()->m_Verbose)
		dbg_msg("server", "Client dropped. ncid=%d addr=%s reason='%s'", ClientID, aAddrStr, pReason);

	if(pThis->m_aClients[ClientID].m_State == CClient::STATE_AUTHED)
		pThis->Main()->OnDelClient(ClientID);
	pThis->m_aClients[ClientID].m_State = CClient::STATE_EMPTY;

	return 0;
}

int CServer::Init(CMain *pMain, const char *Bind, int Port)
{
	m_pMain = pMain;
	m_NetBan.Init();

	for(int i = 0; i < NET_MAX_CLIENTS; i++)
		m_aClients[i].m_State = CClient::STATE_EMPTY;

	m_Ready = false;

	if(Port == 0)
	{
		dbg_msg("server", "Will not bind to port 0.");
		return 1;
	}

	NETADDR BindAddr;
	if(Bind[0] && net_host_lookup(Bind, &BindAddr, NETTYPE_ALL) == 0)
	{
		// got bindaddr
		BindAddr.type = NETTYPE_ALL;
		BindAddr.port = Port;
	}
	else
	{
		mem_zero(&BindAddr, sizeof(BindAddr));
		BindAddr.type = NETTYPE_ALL;
		BindAddr.port = Port;
	}

	if(m_Network.Open(BindAddr, &m_NetBan))
	{
		m_Network.SetCallbacks(NewClientCallback, DelClientCallback, this);
		m_Ready = true;
		dbg_msg("server", "Bound to %s:%d", Bind, Port);
		return 0;
	}
	else
		dbg_msg("server", "Couldn't open socket. Port (%d) might already be in use.", Port);

	return 1;
}

void CServer::Update()
{
	if(!m_Ready)
		return;

	m_NetBan.Update();
	m_Network.Update();

	char aBuf[NET_MAX_PACKETSIZE];
	int ClientID;

	while(m_Network.Recv(aBuf, (int)(sizeof(aBuf))-1, &ClientID))
	{
		dbg_assert(m_aClients[ClientID].m_State != CClient::STATE_EMPTY, "Got message from empty slot.");
		if(m_aClients[ClientID].m_State == CClient::STATE_CONNECTED)
		{
			int ID = -1;
			char aUsername[128] = {0};
			char aPassword[128] = {0};
			const char *pTmp;

			if(!(pTmp = str_find(aBuf, ":"))
				|| (unsigned)(pTmp - aBuf) > sizeof(aUsername) || (unsigned)(str_length(pTmp) - 1) > sizeof(aPassword))
			{
				m_Network.NetBan()->BanAddr(m_Network.ClientAddr(ClientID), 60, "You're an idiot, go away.");
				m_Network.Drop(ClientID, "Fuck off.");
				return;
			}

			str_copy(aUsername, aBuf, pTmp - aBuf + 1);
			str_copy(aPassword, pTmp + 1, sizeof(aPassword));
			if(!*aUsername || !*aPassword)
			{
				m_Network.NetBan()->BanAddr(m_Network.ClientAddr(ClientID), 60, "You're an idiot, go away.");
				m_Network.Drop(ClientID, "Username and password must not be blank.");
				return;
			}

			for(int i = 0; i < NET_MAX_CLIENTS; i++)
			{
				if(!Main()->Client(i)->m_Active)
					continue;

				if(str_comp(Main()->Client(i)->m_aUsername, aUsername) == 0 && str_comp(Main()->Client(i)->m_aPassword, aPassword) == 0)
					ID = i;
			}

			if(ID == -1)
			{
				m_Network.NetBan()->BanAddr(m_Network.ClientAddr(ClientID), 60, "Wrong username and/or password.");
				m_Network.Drop(ClientID, "Wrong username and/or password.");
			}
			else if(Main()->Client(ID)->m_ClientNetID != -1)
			{
				m_Network.Drop(ClientID, "Only one connection per user allowed.");
			}
			else
			{
				m_aClients[ClientID].m_State = CClient::STATE_AUTHED;
				m_aClients[ClientID].m_LastReceived = time_get();
				m_Network.Send(ClientID, "Authentication successful. Access granted.");

				if(m_Network.ClientAddr(ClientID)->type == NETTYPE_IPV4)
					m_Network.Send(ClientID, "You are connecting via: IPv4");
				else if(m_Network.ClientAddr(ClientID)->type == NETTYPE_IPV6)
					m_Network.Send(ClientID, "You are connecting via: IPv6");

				if(Main()->Config()->m_Verbose)
					dbg_msg("server", "ncid=%d authed", ClientID);
				Main()->OnNewClient(ClientID, ID);
			}
		}
		else if(m_aClients[ClientID].m_State == CClient::STATE_AUTHED)
		{
			m_aClients[ClientID].m_LastReceived = time_get();
			if(Main()->Config()->m_Verbose)
				dbg_msg("server", "ncid=%d cmd='%s'", ClientID, aBuf);

			if(str_comp(aBuf, "logout") == 0)
				m_Network.Drop(ClientID, "Logout. Bye Bye ~");
			else
				Main()->HandleMessage(ClientID, aBuf);
		}
	}

	for(int i = 0; i < NET_MAX_CLIENTS; ++i)
	{
		if(m_aClients[i].m_State == CClient::STATE_CONNECTED &&
			time_get() > m_aClients[i].m_TimeConnected + 5 * time_freq())
		{
			m_Network.NetBan()->BanAddr(m_Network.ClientAddr(i), 30, "Authentication timeout.");
			m_Network.Drop(i, "Authentication timeout.");
		}
		else if(m_aClients[i].m_State == CClient::STATE_AUTHED &&
			time_get() > m_aClients[i].m_LastReceived + 15 * time_freq())
			m_Network.Drop(i, "Timeout.");
	}
}

void CServer::Send(int ClientID, const char *pLine)
{
	if(!m_Ready)
		return;

	if(ClientID == -1)
	{
		for(int i = 0; i < NET_MAX_CLIENTS; i++)
		{
			if(m_aClients[i].m_State == CClient::STATE_AUTHED)
				m_Network.Send(i, pLine);
		}
	}
	else if(ClientID >= 0 && ClientID < NET_MAX_CLIENTS && m_aClients[ClientID].m_State == CClient::STATE_AUTHED)
		m_Network.Send(ClientID, pLine);
}

void CServer::Shutdown()
{
	if(!m_Ready)
		return;

	m_Network.Close();
}
