/****************************************************************************
**
** Copyright (C) 2017 The Qt Company Ltd.
** Contact: https://www.qt.io/licensing/
**
** This file is part of the examples of the Qt Wayland module
**
** $QT_BEGIN_LICENSE:BSD$
** Commercial License Usage
** Licensees holding valid commercial Qt licenses may use this file in
** accordance with the commercial license agreement provided with the
** Software or, alternatively, in accordance with the terms contained in
** a written agreement between you and The Qt Company. For licensing terms
** and conditions see https://www.qt.io/terms-conditions. For further
** information use the contact form at https://www.qt.io/contact-us.
**
** BSD License Usage
** Alternatively, you may use this file under the terms of the BSD license
** as follows:
**
** "Redistribution and use in source and binary forms, with or without
** modification, are permitted provided that the following conditions are
** met:
**   * Redistributions of source code must retain the above copyright
**     notice, this list of conditions and the following disclaimer.
**   * Redistributions in binary form must reproduce the above copyright
**     notice, this list of conditions and the following disclaimer in
**     the documentation and/or other materials provided with the
**     distribution.
**   * Neither the name of The Qt Company Ltd nor the names of its
**     contributors may be used to endorse or promote products derived
**     from this software without specific prior written permission.
**
**
** THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
** "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
** LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
** A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
** OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
** SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
** LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
** DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
** THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
** (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
** OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE."
**
** $QT_END_LICENSE$
**
****************************************************************************/

#include "window.h"

#include <QMouseEvent>
#include <QOpenGLWindow>
#include <QOpenGLTexture>
#include <QOpenGLFunctions>
#include <QMatrix4x4>
#include <QTimer>
#include <QDebug>
#include <QJsonDocument>
#include <QJsonObject>

#include "compositor.h"
#include <QtWaylandCompositor/qwaylandseat.h>

Window::Window(QWaylandOutput::Transform transform)
    : m_backgroundTexture(0)
    , m_compositor(0)
    , transform(transform)
    , transformAnimationTimer()
    , suspendAnimationTimer()
{
#if 0
    static int x = 0;
    QTimer *timer = new QTimer(this);
    connect(timer, &QTimer::timeout, this, [this]() {
        qInfo() << "XXX TRANSFORM" << x;
        setTransform((QWaylandOutput::Transform) x & 1 ? QWaylandOutput::Transform90 : QWaylandOutput::Transform270);
        x++;
    });

    timer->start(5000);
#endif

    socketServer = new SocketServer("/run/nubbock/socket", this);

    QObject::connect(socketServer, &SocketServer::jsonReceived, [this](const QJsonObject &obj) {
        const QString transform = obj["transform"].toString();
        if (!transform.isEmpty()) {
            qInfo() << "Transformation change started:" << transform;

            if (transform == "90")
                setTransform(QWaylandOutput::Transform90);

            if (transform == "270")
                setTransform(QWaylandOutput::Transform270);
        }

        if (obj.contains("obj")) {
            const bool suspended = obj["suspended"].toBool();
            setSuspended(suspended);
        }
    });

    socketServer->start();
}

void Window::setCompositor(Compositor *comp) {
    m_compositor = comp;
}

void Window::initializeGL()
{
    QString backgroundImagePath =  QString::fromLocal8Bit(qgetenv("NUBBOCK_BACKGROUND_IMAGE"));

    if (!backgroundImagePath.isEmpty()) {
        QImage backgroundImage = QImage(backgroundImagePath);
        m_backgroundTexture = new QOpenGLTexture(backgroundImage, QOpenGLTexture::DontGenerateMipMaps);
        m_backgroundTexture->setMinificationFilter(QOpenGLTexture::Nearest);
        m_backgroundImageSize = backgroundImage.size();
    }

    QImage black(size(), QImage::Format_Mono);
    black.setColor(0, qRgb(0, 0, 0));
    black.fill(0);

    transformOverlayTexture = new QOpenGLTexture(black, QOpenGLTexture::DontGenerateMipMaps);
    suspendOverlayTexture = new QOpenGLTexture(black, QOpenGLTexture::DontGenerateMipMaps);

    transformAnimationOpacity = 0.0f;
    suspendAnimationOpacity = 0.0f;

    m_textureBlitter.create();
    transformOverlayBlitter.create();
    suspendOverlayBlitter.create();
}

void Window::paintGL()
{
    m_compositor->startRender();
    QOpenGLFunctions *functions = context()->functions();
    functions->glClearColor(.0f, .165f, .31f, 0.5f);
    functions->glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);

    m_textureBlitter.bind();

    QSize sz = size();
    sz.transpose();

    float angle;

    switch (transform) {
    case QWaylandOutput::TransformNormal:
    case QWaylandOutput::TransformFlipped:
        angle = 0.0f;
        break;
    case QWaylandOutput::Transform90:
    case QWaylandOutput::TransformFlipped90:
        angle = 90.0f;
        break;
    case QWaylandOutput::Transform180:
    case QWaylandOutput::TransformFlipped180:
        angle = 180.0f;
        break;
    case QWaylandOutput::Transform270:
    case QWaylandOutput::TransformFlipped270:
        angle = 270.0f;
        break;
    default:
        qWarning() << "Unsupported transform" << transform;
        return;
    }

    QMatrix4x4 targetTransform =
            QOpenGLTextureBlitter::targetTransform(QRect(QPoint(0, 0), m_backgroundImageSize),
                                                   QRect(QPoint(0, 0), sz));
    targetTransform.rotate(angle, 0.0f, 0.0f, 1.0f);

    if (m_backgroundTexture)
        m_textureBlitter.blit(m_backgroundTexture->textureId(),
                              targetTransform,
                              QOpenGLTextureBlitter::OriginTopLeft);

    functions->glEnable(GL_BLEND);
    functions->glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);

    GLenum currentTarget = GL_TEXTURE_2D;
    Q_FOREACH (View *view, m_compositor->views()) {
        if (view->isCursor())
            continue;
        auto texture = view->getTexture();
        if (!texture)
            continue;
        if (texture->target() != currentTarget) {
            currentTarget = texture->target();
            m_textureBlitter.bind(currentTarget);
        }
        QWaylandSurface *surface = view->surface();
        if ((surface && surface->hasContent()) || view->isBufferLocked()) {
            QSize s = view->size();
            if (!s.isEmpty()) {
                QPointF pos = view->position() + view->parentPosition();
                QRectF surfaceGeometry(pos, s);
                auto surfaceOrigin = view->textureOrigin();
                QRectF targetRect(surfaceGeometry.topLeft(), surfaceGeometry.size());
                QMatrix4x4 targetTransform = QOpenGLTextureBlitter::targetTransform(targetRect, QRect(QPoint(), sz));
                targetTransform.rotate(angle, 0.0f, 0.0f, 1.0f);
                m_textureBlitter.blit(texture->textureId(), targetTransform, surfaceOrigin);
            }
        }
    }

    m_textureBlitter.release();

    if (transformAnimationOpacity > 0.0f) {
        transformOverlayBlitter.bind();
        transformOverlayBlitter.setOpacity(transformAnimationOpacity);
        transformOverlayBlitter.blit(transformOverlayTexture->textureId(),
                          targetTransform,
                          QOpenGLTextureBlitter::OriginTopLeft);
        transformOverlayBlitter.release();
    }

    if (suspendAnimationOpacity > 0.0f) {
        suspendOverlayBlitter.bind();
        suspendOverlayBlitter.setOpacity(suspendAnimationOpacity);
        suspendOverlayBlitter.blit(suspendOverlayTexture->textureId(),
                          targetTransform,
                          QOpenGLTextureBlitter::OriginTopLeft);
        suspendOverlayBlitter.release();
    }

    functions->glDisable(GL_BLEND);

    m_compositor->endRender();
}

View *Window::viewAt(const QPointF &point)
{
    View *ret = 0;
    Q_FOREACH (View *view, m_compositor->views()) {
        QRectF geom(view->position(), view->size());
        if (geom.contains(point))
            ret = view;
    }
    return ret;
}

void Window::timerEvent(QTimerEvent *event)
{
    if (event->timerId() == transformAnimationTimer.timerId()) {
        if (transformAnimationUp) {
            transformAnimationOpacity += 0.05;
            if (transformAnimationOpacity >= 1.0f) {
                transformAnimationOpacity = 1.0f;
                transformAnimationUp = false;
                qInfo() << "Transformation change completed:" << transformPending;
                transform = transformPending;
                m_compositor->defaultOutput()->setTransform(transform);
            }
        } else {
            transformAnimationOpacity -= 0.05;
            if (transformAnimationOpacity <= 0.0f) {
                transformAnimationOpacity = 0.0f;
                transformAnimationTimer.stop();
            }
        }
    } else if (event->timerId() == suspendAnimationTimer.timerId()) {
        if (suspendAnimationUp) {
            suspendAnimationOpacity += 0.05;
            if (suspendAnimationOpacity >= 1.0f) {
                suspendAnimationOpacity = 1.0f;
                suspended = true;
                suspendAnimationTimer.stop();
            }
        } else {
            suspendAnimationOpacity -= 0.05;
            if (suspendAnimationOpacity <= 0.0f) {
                suspendAnimationOpacity = 0.0f;
                suspended = false;
                suspendAnimationTimer.stop();
            }
        }
    }

    update();
}

void Window::setTransform(QWaylandOutput::Transform _transform)
{
    if (!transformAnimationTimer.isActive() && transform == _transform)
        return;

    if (transformAnimationTimer.isActive() && transformPending == _transform)
        return;

    transformAnimationTimer.stop();
    transformPending = _transform;
    transformAnimationOpacity = 0.0f;
    transformAnimationUp = true;
    transformAnimationTimer.start(20, this);
}

void Window::setSuspended(bool _suspended)
{
    if (suspended == _suspended)
        return;

    suspendAnimationTimer.stop();
    suspendAnimationUp = _suspended;
    suspendAnimationTimer.start(20, this);
}

QPointF Window::transformPosition(const QPointF p)
{
    qreal x = p.x();
    qreal y = p.y();

    switch (transform) {
    case QWaylandOutput::TransformNormal:
        x = p.x();
        y = p.y();
        break;
    case QWaylandOutput::TransformFlipped:
        x = p.x();
        y = size().height() - p.y();
        break;
    case QWaylandOutput::Transform90:
        x = size().height() - p.y();
        y = p.x();
        break;
    case QWaylandOutput::Transform180:
        x = p.x();
        y = size().height() - p.y();
        break;
    case QWaylandOutput::Transform270:
        x = p.y();
        y = size().width() - p.x();
        break;
    case QWaylandOutput::TransformFlipped90:
    case QWaylandOutput::TransformFlipped180:
    case QWaylandOutput::TransformFlipped270:
        /* TODO */
        break;
    }

    return QPointF(x, y);
}

void Window::mousePressEvent(QMouseEvent *e)
{
    QPointF p = transformPosition(e->localPos());

    if (m_mouseView.isNull()) {
        m_mouseView = viewAt(p);
        if (!m_mouseView) {
            return;
        }

        m_initialMousePos = p;
        m_mouseOffset = p - m_mouseView->position();

        QMouseEvent moveEvent(QEvent::MouseMove, p, e->globalPos(), Qt::NoButton, Qt::NoButton, e->modifiers());
        sendMouseEvent(&moveEvent, p, m_mouseView);
    }

    sendMouseEvent(e, p, m_mouseView);
}

void Window::mouseReleaseEvent(QMouseEvent *e)
{
    if (e->buttons() == Qt::NoButton)
        m_mouseView = 0;
}

void Window::mouseMoveEvent(QMouseEvent *e)
{
    QPointF p = transformPosition(e->localPos());
    View *view = m_mouseView ? m_mouseView.data() : viewAt(p);
    sendMouseEvent(e, p, view);
}

void Window::touchEvent(QTouchEvent *e)
{
    QWaylandSeat *input = m_compositor->defaultSeat();

    const QList<QTouchEvent::TouchPoint> points = e->touchPoints();
    if (points.isEmpty())
        return;

    QPointF p = transformPosition(points.at(0).normalizedPos());
    View *view = viewAt(p);

    if (!view)
        return;

    QWaylandSurface *surface = view->surface();

    if (e->type() == QEvent::TouchCancel) {
        input->sendTouchCancelEvent(surface->client());
        return;
    }

//    QtWayland::TouchExtensionGlobal *ext = QtWayland::TouchExtensionGlobal::findIn(m_compositor);
//    if (ext && ext->postTouchEvent(event, surface))
//        return;

    for (int i = 0; i < points.count(); ++i) {
        const QTouchEvent::TouchPoint tp(points.at(i));
        const QPointF pos = transformPosition(tp.pos());

        // Convert the local pos in the compositor window to surface-relative.
        //input->sendTouchPointEvent(surface, tp.id(), pos, tp.state());
    }

    input->sendTouchFrameEvent(surface->client());
}

void Window::sendMouseEvent(QMouseEvent *e, QPointF p, View *target)
{
    // At this point, the mouse event is already transformed

    QPointF mappedPos = e->localPos();
    if (target)
        mappedPos -= target->position();
    QMouseEvent viewEvent(e->type(), mappedPos, p, e->button(), e->buttons(), e->modifiers());
    m_compositor->handleMouseEvent(target, &viewEvent);
}

void Window::keyPressEvent(QKeyEvent *e)
{
    m_compositor->defaultSeat()->sendKeyPressEvent(e->nativeScanCode());
}

void Window::keyReleaseEvent(QKeyEvent *e)
{
    m_compositor->defaultSeat()->sendKeyReleaseEvent(e->nativeScanCode());
}
