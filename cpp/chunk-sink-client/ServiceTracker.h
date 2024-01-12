#pragma once

#include <avahi-client/client.h>
#include <avahi-client/lookup.h>

#include <avahi-common/simple-watch.h>
#include <avahi-common/malloc.h>
#include <avahi-common/error.h>

#include <memory>
#include <thread>
#include <mutex>
#include <map>

struct ServiceEntry {
    std::string hostName;
    uint16_t port;
    std::string address;
    std::string txt;
};

class ServiceTracker
{
public:
    ServiceTracker(const ServiceTracker&) = delete;
    ServiceTracker& operator=(const ServiceTracker&) = delete;
    ServiceTracker(const std::string &service);

    std::map<std::string, std::map<std::string,ServiceEntry>> GetServiceMap();

private:
    static void resolveCB(AvahiServiceResolver *r, AvahiIfIndex interface, AvahiProtocol protocol, AvahiResolverEvent event,
                   const char *name, const char *type, const char *domain, const char *host_name, const AvahiAddress *address, 
                   uint16_t port, AvahiStringList *txt, AvahiLookupResultFlags flags, void *userdata);

    static void browseCB(AvahiServiceBrowser *b, AvahiIfIndex interface, AvahiProtocol protocol, AvahiBrowserEvent event, 
                         const char *name, const char *type, const char *domain, AvahiLookupResultFlags flags, void *userdata);

    static void clientCB(AvahiClient *c, AvahiClientState state, void * userdata);

    void run(const std::string &serviceName);
    void addEntry(const std::string &key, const ServiceEntry &e);
    void removeEntry(const std::string &key);
    void printMap();

    std::unique_ptr<std::thread> m_worker;
    // First key is hostname, second key is address:port
    std::map<std::string, std::map<std::string,ServiceEntry>> m_serviceMap;
    std::mutex m_lock;
};

struct UserData {
    AvahiClient *avahiClient;
    ServiceTracker *serviceTracker;
};