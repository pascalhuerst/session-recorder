#include "socketserver.h"
#include <QDebug>
#include <QLocalSocket>
#include <QJsonDocument>
#include <QFile>

SocketServer::SocketServer(const QString &path, QObject *parent) :
    QObject(parent),
    path(path),
    localServer(this)
{
    QObject::connect(&localServer, &QLocalServer::newConnection, this, [this]() {
        QLocalSocket *socketClient = localServer.nextPendingConnection();

        if (!socketClient)
            return;

        QObject::connect(socketClient, &QLocalSocket::readyRead, [this, socketClient]() {
            QByteArray buf = socketClient->readAll();
            QList<QByteArray> messages = buf.split(0);

            foreach (QByteArray message, messages) {
                QJsonDocument doc = QJsonDocument::fromJson(message);

                if (!doc.isObject())
                    return;

                QJsonObject obj = doc.object();
                emit jsonReceived(obj);
            }
        });
    });
}

bool SocketServer::start()
{
    localServer.setMaxPendingConnections(1);
    QFile::remove(path);
    if (localServer.listen(path)) {
        qInfo() << "Listening on" << localServer.serverName();
        return true;
    }

    qWarning() << "Error listening on" << localServer.serverName() << ":" << localServer.serverError();
    return false;
}
