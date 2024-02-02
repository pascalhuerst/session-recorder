#ifndef SOCKETSERVER_H
#define SOCKETSERVER_H

#include <QObject>
#include <QLocalServer>
#include <QJsonObject>

class SocketServer : public QObject
{
    Q_OBJECT
public:
    explicit SocketServer(const QString &path, QObject *parent = nullptr);
    bool start();

signals:
    void jsonReceived(const QJsonObject &obj);

public slots:

private:
    QString path;
    QLocalServer localServer;
};

#endif // SOCKETSERVER_H
