#!/etc/bin/env python
# -*- coding: utf-8 -*-

import uuid
import etcd3
import logging
import simplejson as json

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


class Tag(object):

    @property
    def name(self):
        if hasattr(self, "_tag"):
            return self._tag
        else:
            return ""

    @property
    def bytes(self):
        if hasattr(self, "value"):
            return bytes(self.value)
        else:
            return bytes()


class String(str, Tag):

    def __init__(self, tag: str):
        super(String, self).__init__()
        self._tag = tag

    @property
    def value(self):
        return self.__str__()


class Slice(list, Tag):

    def __init__(self, tag: str):
        super(Slice, self).__init__()
        self._tag = tag

    @property
    def value(self):
        new = list()
        for i in self:
            new.append(i)
        return new

    @property
    def bytes(self):
        return bytes(json.dumps(self.value))


class Integer(int, Tag):

    def __init__(self, tag: str):
        super(Integer, self).__init__()
        self._tag = tag

    @property
    def value(self):
        return self


class Boolean(bool, Tag):

    def __init__(self, tag: str):
        super(Boolean, self).__init__()
        self._tag = tag

    @property
    def bytes(self):
        return bytes(1) if self is True else bytes(0)


class ServiceDefinition(GatewayDefinition):
    id = String("ID")
    name = String("Name")
    node = Slice("Node")


class HealthCheckDefinition(GatewayDefinition):
    id = String("ID")
    path = String("Path")
    timeout = Integer("Timeout")
    interval = Integer("Interval")
    retry = Boolean("Retry")
    retry_time = Integer("RetryTime")


class NodeDefinition(GatewayDefinition):
    id = String("ID")
    name = String("Name")
    status = Integer("Status")
    health_check = String("HealthCheck")
    service = String("Service")


class RouterDefinition(GatewayDefinition):
    id = String("ID")
    name = String("Name")
    frontend = String("FrontendApi")
    backend = String("BackendApi")
    service = String("Service")


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

        self._register_node()
        self._register_service()
        self._register_router()
        self._register_hc()

    def _register_node(self):
        if self._client is None or not isinstance(self._client, etcd3.Etcd3Client):
            raise ClientNotExist

        n = self._node
        c = self._client

        def put(cli, node_id, attr):
            cli.put(ROOT + SLASH.join([NODE_KEY, NODE_PREFIX + node_id, attr.name]), attr.bytes)

        v = c.get(ROOT + SLASH.join([NODE_KEY, NODE_PREFIX + n.id, n.id.name]))
        if n.id == v:
            # node exists, update node
            pass
        else:
            # node not exists, put new one
            put(c, n.id, n.id)
        put(c, n.id, n.name)
        put(c, n.id, n.status)
        put(c, n.id, n.health_check)
        put(c, n.id, n.service)

    def _register_service(self):
        if self._client is None or not isinstance(self._client, etcd3.Etcd3Client):
            raise ClientNotExist

        s = self._service
        c = self._client

        def put(cli, service_id, attr):
            cli.put(ROOT + SLASH.join([SERVICE_KEY, SERVICE_PREFIX + service_id, attr.name]), attr.bytes)

        v = c.get(ROOT + SLASH.join([SERVICE_KEY, SERVICE_PREFIX + s.id, s.id.name]))
        if s.id == v:
            # service exists, check node exists in service.node_slice
            node_slice_json = c.get(ROOT + SLASH.join([SERVICE_KEY, SERVICE_PREFIX + s.id, s.node.name]))
            try:
                node_slice = json.loads(node_slice_json)
            except Exception as e:
                logging.exception(e)
                node_slice = list()

            flag = 0
            for n in node_slice:
                if n == self._node.id:
                    flag = 1
                    break
            if flag == 0 and len(node_slice) > 0:
                node_slice.append(self._node.id)
                c.put(ROOT + SLASH.join([SERVICE_KEY, SERVICE_PREFIX + s.id, s.node.name]),
                      bytes(json.dumps(node_slice)))
        else:
            # service not exits, put new one
            put(c, s.id, s.id)
            put(c, s.id, s.name)

    def _register_router(self):
        if self._client is None or not isinstance(self._client, etcd3.Etcd3Client):
            raise ClientNotExist

        c = self._client
        rl = self._router

        def put(cli, router_id, attr):
            cli.put(ROOT + SLASH.join([ROUTER_KEY, ROUTER_PREFIX + router_id, attr.name]), attr.bytes)

        for r in rl:
            v = c.get(ROOT + SLASH.join([ROUTER_KEY, ROUTER_PREFIX + r.id, r.id.name]))
            if r.id == v:
                # router exists, update
                pass
            else:
                put(c, r.id, r.id)

            put(c, r.id, r.name)
            put(c, r.id, r.service)
            put(c, r.id, r.frontend)
            put(c, r.id, r.backend)

    def _register_hc(self):
        if self._client is None or not isinstance(self._client, etcd3.Etcd3Client):
            raise ClientNotExist

        c = self._client
        hc = self._hc

        def put(cli, hc_id, attr):
            cli.put(ROOT + SLASH.join([HEALTH_CHECK_KEY, HEALTH_CHECK_PREFIX + hc_id, attr.name]), attr.bytes)

        v = c.get(ROOT + SLASH.join([HEALTH_CHECK_KEY, HEALTH_CHECK_PREFIX + hc.id, hc.id.name]))
        if hc.id == v:
            # health check exists, update
            pass
        else:
            put(c, hc.id, hc.id)

        put(c, hc.id, hc.path)
        put(c, hc.id, hc.interval)
        put(c, hc.id, hc.timeout)
        put(c, hc.id, hc.retry)
        put(c, hc.id, hc.retry_time)
