#include <stdio.h>
#include <assert.h>
#include <stdlib.h>
#include <time.h>
#include <thread>
#include <iostream>

#include "ServiceTracker.h"

#include <avahi-client/client.h>
#include <avahi-client/lookup.h>

#include <avahi-common/simple-watch.h>
#include <avahi-common/malloc.h>
#include <avahi-common/error.h>

static AvahiSimplePoll *simple_poll = NULL;

std::string makeKey(const std::string &name, const std::string &type, const std::string &domain, int interface, int protocol)
{
    char keyBuf[64];
    snprintf(keyBuf, 64, "%s:%s:%s:%d:%d", name.c_str(), type.c_str(), domain.c_str(), interface, protocol);
    std::cout << "KEY: " << keyBuf << std::endl;
    return keyBuf;
}


ServiceTracker::ServiceTracker(const std::string &serviceName)
{
    run(serviceName);
}

void ServiceTracker::run(const std::string &serviceName)
{
     m_worker.reset(new std::thread([=] {

        int error;

        if (!(simple_poll = avahi_simple_poll_new())) {
            std::cerr << "Failed to create simple poll object." << std::endl;
            return;
        }

        auto client = avahi_client_new(avahi_simple_poll_get(simple_poll), AvahiClientFlags(0), ServiceTracker::clientCB, static_cast<void *>(this), &error);
        if (!client) {
            std::cerr << "Failed to create client: " << avahi_strerror(error) << std::endl;
            avahi_simple_poll_free(simple_poll);
        }

        UserData ud = { client, this };

        printf("[%s]\n", serviceName.c_str());
        auto sb = avahi_service_browser_new(client, AVAHI_IF_UNSPEC, AVAHI_PROTO_UNSPEC, serviceName.c_str(), NULL, AvahiLookupFlags(0), ServiceTracker::browseCB, reinterpret_cast<void *>(&ud));
        if (!sb) {
            std::cerr << "Failed to create service browser: " << avahi_strerror(avahi_client_errno(client)) << std::endl;
            avahi_client_free(client);
            avahi_simple_poll_free(simple_poll);
            return;
        }

        avahi_simple_poll_loop(simple_poll);
    }));
}

void ServiceTracker::addEntry(const std::string &key, const ServiceEntry &e)
{
    std::lock_guard<std::mutex> guard(this->m_lock);
    m_serviceMap[e.hostName][key] = e;
}

void ServiceTracker::removeEntry(const std::string &key)
{
    std::lock_guard<std::mutex> guard(this->m_lock);
    for (auto &hostMap : m_serviceMap) {
        hostMap.second.erase(key);
    }
}

ServiceTracker::ServiceMap ServiceTracker::GetServiceMap()
{
    std::lock_guard<std::mutex> guard(this->m_lock);
    return m_serviceMap;
}

void ServiceTracker::printMap()
{
    for (auto &hostMap : m_serviceMap) {
        printf("%s\n", hostMap.first.c_str());

        for (auto &server : hostMap.second) {
            printf("  %s:  %s\n", server.first.c_str(), server.second.address.c_str());
        }
    }
}


// Static
void ServiceTracker::resolveCB(AvahiServiceResolver *r, AvahiIfIndex interface, AvahiProtocol protocol, AvahiResolverEvent event,
                             const char *name, const char *type, const char *domain, const char *host_name, const AvahiAddress *address, 
                             uint16_t port, AvahiStringList *txt, AvahiLookupResultFlags flags, void *userdata)
{

    if (!r) {
        return;
    }

    UserData *ud = reinterpret_cast<UserData *>(userdata);

    switch (event) {
        case AVAHI_RESOLVER_FAILURE:
            std::cerr << "(Resolver) Failed to resolve service '" << name 
                      << "' of type '" << type 
                      << "' in domain '" << domain 
                      << "': " << avahi_strerror(avahi_client_errno(avahi_service_resolver_get_client(r))) << std::endl;
            break;

        case AVAHI_RESOLVER_FOUND: {
            char a[AVAHI_ADDRESS_STR_MAX], *t;

            avahi_address_snprint(a, sizeof(a), address);
            t = avahi_string_list_to_string(txt);

            auto key = makeKey(name, type, domain, interface, protocol); 
            ud->serviceTracker->addEntry(key, ServiceEntry {
                .hostName = host_name,
                .port = port,
                .address = a,
                .txt = t
            });

            avahi_free(t);
        }
    }
    avahi_service_resolver_free(r);
}

// Static
void ServiceTracker::browseCB(AvahiServiceBrowser *b, AvahiIfIndex interface, AvahiProtocol protocol, AvahiBrowserEvent event, 
                            const char *name, const char *type, const char *domain, AvahiLookupResultFlags flags, void *userdata)
{
    UserData *ud = reinterpret_cast<UserData *>(userdata);

    if (!b) {
        return;
    }

    /* Called whenever a new services becomes available on the LAN or is removed from the LAN */

    switch (event) {
        case AVAHI_BROWSER_FAILURE:
            std::cerr << "(Browser) " << avahi_strerror(avahi_client_errno(avahi_service_browser_get_client(b))) << std::endl;
            avahi_simple_poll_quit(simple_poll);
            return;

        case AVAHI_BROWSER_NEW:
            if (!(avahi_service_resolver_new(ud->avahiClient, interface, protocol, name, type, domain, AVAHI_PROTO_UNSPEC, AvahiLookupFlags(0), ServiceTracker::resolveCB, userdata)))
                std::cerr << "Failed to resolve service '" << name << "': " << avahi_strerror(avahi_client_errno(ud->avahiClient)) << std::endl;
            break;

        case AVAHI_BROWSER_REMOVE:
            ud->serviceTracker->removeEntry(makeKey(name, type, domain, interface, protocol));
            break;

        case AVAHI_BROWSER_ALL_FOR_NOW:
        case AVAHI_BROWSER_CACHE_EXHAUSTED:
            fprintf(stderr, "(Browser) %s\n", event == AVAHI_BROWSER_CACHE_EXHAUSTED ? "CACHE_EXHAUSTED" : "ALL_FOR_NOW");
            break;
    }
}

// Static
void ServiceTracker::clientCB(AvahiClient *c, AvahiClientState state, void * userdata)
{
    //ServiceTracker *sd = reinterpret_cast<ServiceTracker *>(userdata);

    if (state == AVAHI_CLIENT_FAILURE) {
        std::cerr << "Server connection failure, quitting:  " << avahi_strerror(avahi_client_errno(c));
        avahi_simple_poll_quit(simple_poll);
    }

}