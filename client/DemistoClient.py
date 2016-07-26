#!/usr/bin/env python2.7
from requests import Session
from requests.packages.urllib3.exceptions import InsecureRequestWarning
from requests.packages.urllib3 import disable_warnings
disable_warnings(InsecureRequestWarning)

class Client:
    XSRF_TOKEN_KEY = "X-XSRF-TOKEN"
    XSRF_COOKIE_KEY = "XSRF-TOKEN"
    
    # New client that does not do anything yet before the login
    def __init__(self, username, password, server):
        if not (username and password and server):
            raise ValueError("You must provide all three parameters")
        if not server[-1] == '/':
            server += '/'
        try:
            s = Session()
            r = s.get(server,verify=False)
        except InsecureRequestWarning:
            pass
        self.token = r.cookies[Client.XSRF_COOKIE_KEY]
        self.username = username
        self.password = password
        self.server = server
        self.session = s

    def req(self, method, path, contentType, data):
        h = { "Accept": "application/json",
                    "Content-type" : contentType if contentType else "application/json",
                    Client.XSRF_TOKEN_KEY: self.token }

        try:
            if self.session:
                r = self.session.request(method, self.server+path, headers=h, verify=False, json=data)
            else:
                raise RuntimeError("Session not initialized!")
        except InsecureRequestWarning:
            pass
        return r
        
    def Login(self):
        data = {'user': self.username,
                'password': self.password}
             
        return self.req("POST","login","",data)
        #self.sessionCookie = r.cookies['S']

    def Logout(self):
        return self.req("POST","logout","",{})

    def NewIncidentExample(self):
        data = {"type":"Malware",
                "name": "Test Incident",
                "owner": "lior",
                "severity": 2,
                "labels": [{"type":"label1", "value": "value1"}],
                "details": "Some incident details"}
        
        return self.req("POST","incident","",data).content
