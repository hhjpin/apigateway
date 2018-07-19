#!/etc/bin/env python
# -*- coding: utf-8 -*-

import etcd3
import logging

"""
register config ex:
    {
        "service": "",
        "server": {
            "name": "",
            "health_check": {
                "path": "/check",
                "timeout": 0,        
                "interval": 0,      
                "retry": 0,         
                "retry_time": 0
            }
        },
        "router": [
            {
                "name": "",
                "outer_api": "/api/v1/ds/cart",
                "inner_api": "/v1/cart",
            }
        ]
    }
"""

class ClientNotExist(Exception): pass
class PendingRegisterHandlerNotExist(Exception): pass


class EtcdConf(object):

    host = str()
    port = int()
    user = str()
    password = str()


class PendingRouter(object):




class ApiGatewayRegistrant(object):

    def __init__(self):
        self._client = None
        self._r = None

    @classmethod
    def initialize(cls, etcd_conf: EtcdConf, pending_router: list):
        ec = etcd_conf
        self = cls()
        try:
            self._client = etcd3.client(host=ec.host, port=ec.port, user=ec.user, password=ec.password)
        except Exception as e:
            logging.exception(str(e))
            self._client = None
        self._r = pending_router
        return self

    def set_health_check(self):

    def register(self):

        if self._client is None:
            raise ClientNotExist
        if self._r is None:
            raise PendingRegisterHandlerNotExist

    def _register
