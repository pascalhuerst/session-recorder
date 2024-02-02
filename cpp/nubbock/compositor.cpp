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

#include "compositor.h"

#include <QMouseEvent>
#include <QKeyEvent>
#include <QTouchEvent>

#include <QtWaylandCompositor/QWaylandWlShellSurface>
#include <QtWaylandCompositor/qwaylandseat.h>
#include <QtWaylandCompositor/qwaylanddrag.h>

#include <QDebug>
#include <QOpenGLContext>

#ifndef GL_TEXTURE_EXTERNAL_OES
#define GL_TEXTURE_EXTERNAL_OES 0x8D65
#endif

View::View(Compositor *compositor)
    : m_compositor(compositor)
    , m_textureTarget(GL_TEXTURE_2D)
    , m_texture(0)
    , m_parentView(nullptr)
{}

QOpenGLTexture *View::getTexture()
{
    if (advance()) {
        QWaylandBufferRef buf = currentBuffer();
        m_texture = buf.toOpenGLTexture();
        if (surface()) {
            //m_size = surface()->size();
            m_origin = buf.origin() == QWaylandSurface::OriginTopLeft
                    ? QOpenGLTextureBlitter::OriginTopLeft
                    : QOpenGLTextureBlitter::OriginBottomLeft;
        }
    }

    return m_texture;
}

QOpenGLTextureBlitter::Origin View::textureOrigin() const
{
    return m_origin;
}

QSize View::size() const
{
    //return surface() ? surface()->size() : m_size;
    return m_size;
}

bool View::isCursor() const
{
    return surface() && surface()->isCursorSurface();
}

void View::onOffsetForNextFrame(const QPoint &offset)
{
    m_offset = offset;
    setPosition(position() + offset);
}

Compositor::Compositor(QWindow *window)
    : QWaylandCompositor()
    , m_window(window)
{
}

Compositor::~Compositor()
{
}

void Compositor::create()
{
    QWaylandOutput *output = new QWaylandOutput(this, m_window);
    QWaylandOutputMode mode(QSize(1280, 800), 60000);
    output->addMode(mode, true);
    QWaylandCompositor::create();
    output->setCurrentMode(mode);

    setDefaultOutput(output);

    output->setTransform(QWaylandOutput::Transform270);

    connect(this, &QWaylandCompositor::surfaceCreated, this, &Compositor::onSurfaceCreated);
    connect(defaultSeat(), &QWaylandSeat::cursorSurfaceRequest, this, &Compositor::adjustCursorSurface);

    connect(this, &QWaylandCompositor::subsurfaceChanged, this, &Compositor::onSubsurfaceChanged);
}

void Compositor::onSurfaceCreated(QWaylandSurface *surface)
{
    connect(surface, &QWaylandSurface::surfaceDestroyed, this, &Compositor::surfaceDestroyed);
    connect(surface, &QWaylandSurface::hasContentChanged, this, &Compositor::surfaceHasContentChanged);
    connect(surface, &QWaylandSurface::redraw, this, &Compositor::triggerRender);
    connect(surface, &QWaylandSurface::subsurfacePositionChanged, this, &Compositor::onSubsurfacePositionChanged);

    View *view = new View(this);
    view->setSurface(surface);
    view->setOutput(outputFor(m_window));

    outputFor(m_window)->setTransform(QWaylandOutput::Transform270);

    m_views << view;
    connect(surface, &QWaylandSurface::offsetForNextFrame, view, &View::onOffsetForNextFrame);
}

void Compositor::surfaceHasContentChanged()
{
    QWaylandSurface *surface = qobject_cast<QWaylandSurface *>(sender());
    if (surface->hasContent()) {
        if (surface->role() == QWaylandWlShellSurface::role())
            defaultSeat()->setKeyboardFocus(surface);
    }
    triggerRender();
}

void Compositor::surfaceDestroyed()
{
    QWaylandSurface *surface = qobject_cast<QWaylandSurface*>(sender());

    // purge all views with no surface
    while (true) {
        View *view = findView(Q_NULLPTR);
        if (!view)
            break;

        m_views.removeAll(view);
        delete view;
    }

    triggerRender();
}

View * Compositor::findView(const QWaylandSurface *s) const
{
    Q_FOREACH (View* view, m_views) {
        if (view->surface() == s)
            return view;
    }
    return Q_NULLPTR;
}

void Compositor::onSubsurfaceChanged(QWaylandSurface *child, QWaylandSurface *parent)
{
    View *view = findView(child);
    View *parentView = findView(parent);
    view->setParentView(parentView);
}

void Compositor::onSubsurfacePositionChanged(const QPoint &position)
{
    QWaylandSurface *surface = qobject_cast<QWaylandSurface*>(sender());
    if (!surface)
        return;
    View *view = findView(surface);
    view->setPosition(position);
    triggerRender();
}

void Compositor::triggerRender()
{
    m_window->requestUpdate();
}

void Compositor::startRender()
{
    QWaylandOutput *out = defaultOutput();
    if (out)
        out->frameStarted();
}

void Compositor::endRender()
{
    QWaylandOutput *out = defaultOutput();
    if (out)
        out->sendFrameCallbacks();
}

void Compositor::updateCursor()
{
    m_cursorView.advance();
    QImage image = m_cursorView.currentBuffer().image();
    if (!image.isNull())
        m_window->setCursor(QCursor(QPixmap::fromImage(image), m_cursorHotspotX, m_cursorHotspotY));
}

void Compositor::adjustCursorSurface(QWaylandSurface *surface, int hotspotX, int hotspotY)
{
    if ((m_cursorView.surface() != surface)) {
        if (m_cursorView.surface())
            disconnect(m_cursorView.surface(), &QWaylandSurface::redraw, this, &Compositor::updateCursor);
        if (surface)
            connect(surface, &QWaylandSurface::redraw, this, &Compositor::updateCursor);
    }

    m_cursorView.setSurface(surface);
    m_cursorHotspotX = hotspotX;
    m_cursorHotspotY = hotspotY;

    if (surface && surface->hasContent())
        updateCursor();
}

void Compositor::handleMouseEvent(QWaylandView *target, QMouseEvent *me)
{
    QWaylandSeat *input = defaultSeat();
    QWaylandSurface *surface = target ? target->surface() : nullptr;
    switch (me->type()) {
        case QEvent::MouseButtonPress:
            input->sendMousePressEvent(me->button());
            if (surface != input->keyboardFocus()) {
                if (surface == nullptr || surface->role() == QWaylandWlShellSurface::role())
                    input->setKeyboardFocus(surface);
            }
            break;
    case QEvent::MouseButtonRelease:
         input->sendMouseReleaseEvent(me->button());
         break;
    case QEvent::MouseMove:
        input->sendMouseMoveEvent(target, me->localPos(), me->globalPos());
    default:
        break;
    }
}

void Compositor::handleTouchEvent(QWaylandView *target, QTouchEvent *e)
{
    QWaylandSurface *surface = target ? target->surface() : nullptr;
    QWaylandSeat *input = defaultSeat();

    input->sendFullTouchEvent(surface, e);
}
