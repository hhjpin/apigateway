#!/etc/bin/env python
# -*- coding: utf-8 -*-

import uuid
import etcd3
import logging

from .const import *

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


class ClientNotExist(Exception):
    pass


class PredefinedTypeError(TypeError):
    pass


class NotDefinedAttributeError(Exception):
    pass


class GatewayDefinition(object):

    def __setattr__(self, key, value):
        if key in self.__dict__:
            if isinstance(self.__annotations__[key], value):
                self.__setattr__(key, value)
            else:
                raise PredefinedTypeError
        else:
            raise NotDefinedAttributeError


class ServiceDefinition(GatewayDefinition):
    id: str
    name: str
    node: list


class HealthCheckDefinition(GatewayDefinition):
    id: str
    path: str
    timeout: int
    interval: int
    retry: bool
    retry_time: int


class NodeDefinition(GatewayDefinition):
    id: str
    name: str
    status: int
    health_check: str
    service: str


class RouterDefinition(GatewayDefinition):
    id: str
    name: str
    frontend: str
    backend: str
    service: str


class EtcdConf(object):
    host: str
    port: int
    user: str
    password: str

    def __str__(self):
        return "HOST: {}, PORT: {}, USER: {}, PASSWORD: {}".format(self.host, str(self.port), self.user, self.password)

# class PendingRouter(object):


class ApiGatewayRegistrant(object):

    def __init__(self, service: ServiceDefinition, node: NodeDefinition,
                 hc: HealthCheckDefinition, etcd_conf: EtcdConf, router: list):
        self._client = None
        self._router = list()

        uid = uuid.getnode()
        hc.id = uid
        node.id = uid
        node.name += uid
        node.status = 0
        service.id = uid
        service.name += uid
        self._hc = hc
        self._node = node
        self._service = service

        ec = etcd_conf
        try:
            self._client = etcd3.client(host=ec.host, port=ec.port, user=ec.user,
                                        password=ec.password)
        except Exception as e:
            logging.exception(str(e))
            self._client = None
        for r in router:
            if not isinstance(r, RouterDefinition):
                raise PredefinedTypeError
            self._router.append(r)

    def register(self):

        if self._client is None:
            raise ClientNotExist

    def _register_node(self):
        if self._client is None or not isinstance(self._client, etcd3.Etcd3Client):
            raise ClientNotExist

        self._client.get("")
