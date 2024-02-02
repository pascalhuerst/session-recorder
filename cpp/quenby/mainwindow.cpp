/***
  Copyright (c) 2017 Nepos GmbH

  Authors: Daniel Mack <daniel@nepos.io>

  This program is free software; you can redistribute it and/or
  modify it under the terms of the GNU General Public License
  as published by the Free Software Foundation; either version 2
  of the License, or (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with this program; if not, see <http://www.gnu.org/licenses/>.
***/

#include <QWebEngineProfile>
#include <QWebEngineView>
#include <QWebEngineHistory>
#include <QQuickItem>
#include <QQuickWidget>
#include <QVBoxLayout>
#include <QApplication>
#include <QTemporaryFile>
#include <QTimer>
#include <QWebEngineProfile>

#include "mainwindow.h"
#include "webenginepage.h"

MainWindow::MainWindow(QUrl mainViewUrl, int mainViewWidth, int mainViewHeight, QWidget *parent) :
    QMainWindow(parent),
    keyboardEnabled(true),
    frame(new QWidget(this)),
    browserWidget(new QWidget(frame)),
    windowLayout(new QVBoxLayout(this)),
    browserLayout(new QVBoxLayout(this)),
    quickWidget(new QQuickWidget(frame)),
    inputPanel(nullptr),
    controlChannel(),
    controlInterface(),
    views()
{
    setCentralWidget(frame);
    frame->setLayout(windowLayout);

    setAttribute(Qt::WA_TranslucentBackground);

    QWebEngineProfile *profile = QWebEngineProfile::defaultProfile();
    profile->clearHttpCache();
    profile->setHttpCacheType(QWebEngineProfile::DiskHttpCache);

    browserWidget->setLayout(browserLayout);
    browserWidget->resize(mainViewWidth, mainViewHeight);

    quickWidget->setFocusPolicy(Qt::NoFocus);
    quickWidget->setSource(QUrl("qrc:/inputpanel.qml"));
    quickWidget->setSizePolicy(QSizePolicy::Fixed, QSizePolicy::Fixed);
    quickWidget->setVisible(false);

    windowLayout->addWidget(browserWidget);
    windowLayout->addWidget(quickWidget);
    windowLayout->setSpacing(0);
    //windowLayout->setMargin(0);

    browserLayout->setSpacing(0);
    //browserLayout->setMargin(0);

    inputPanel = quickWidget->rootObject();
    if (inputPanel)
        inputPanel->setProperty("width", mainViewWidth);

    connect(inputPanel, SIGNAL(activated(bool)), this, SLOT(onKeyboardActiveChanged(bool)));
    connect(inputPanel, SIGNAL(heightChanged(int)), this, SLOT(onKeyboardHeightChanged(int)));

    auto key = nextKey();
    auto *view = addWebView(key);
    // Add default browser to layout, so it scales when keyboard is shown
    browserLayout->addWidget(view);
    auto *page = view->page();
    page->setBackgroundColor(Qt::transparent);
    view->page()->setWebChannel(&controlChannel);
    view->setUrl(mainViewUrl);
    view->setGeometry(0, 0, mainViewWidth, mainViewHeight);
    view->setAutoFillBackground(false);

    createControlInterface();

    QObject::connect(page, &WebEnginePage::featurePermissionRequested, [this, page](const QUrl &securityOrigin, QWebEnginePage::Feature feature) {

        enum QWebEnginePage::PermissionPolicy verdict = QWebEnginePage::PermissionDeniedByUser;

        if (securityOrigin.host() == "localhost" ||
                (securityOrigin.host() == "localhost:3000" && (feature == QWebEnginePage::MediaAudioVideoCapture ||
                                                               feature == QWebEnginePage::MediaVideoCapture))) {
            verdict = QWebEnginePage::PermissionGrantedByUser;
        }

        qInfo() << "WebEnginePage::featurePermissionRequested: " << feature << " verdict " << verdict;
        page->setFeaturePermission(securityOrigin, feature, verdict);

    });

    QObject::connect(view, &QWebEngineView::titleChanged, [this](const QString &title) {
        setWindowTitle(title);
    });
}

MainWindow::~MainWindow()
{
    if (quickWidget)
        delete quickWidget;
}

void MainWindow::setKeyboardEnabled(bool enabled)
{
    keyboardEnabled = enabled;
}

void MainWindow::resizeEvent(QResizeEvent *event)
{
    QMainWindow::resizeEvent(event);

    if (inputPanel)
        inputPanel->setProperty("width", event->size().width());
}

void MainWindow::onKeyboardActiveChanged(bool a)
{
    if (!keyboardEnabled)
        return;

    quickWidget->setVisible(a);
    if (a) {
        emit controlInterface.onKeyboardShown(quickWidget->size().height());
        quickWidget->grabMouse();
    } else {
        emit controlInterface.onKeyboardHidden();
        quickWidget->releaseMouse();
    }
}

void MainWindow::onKeyboardHeightChanged(int h)
{
    auto newSize = quickWidget->size();
    newSize.setHeight(h);
    quickWidget->resize(newSize);
}

void MainWindow::createControlInterface()
{
    QObject::connect(&controlInterface, &ControlInterface::onCreateWebViewRequest, [this]() {
        auto key = nextKey();
        QWebEngineView *view = addWebView(key);
        WebEnginePage *page = static_cast<WebEnginePage*>(view->page());

        QObject::connect(view, &QWebEngineView::urlChanged, [this, key](const QUrl &url) {
            emit controlInterface.onWebViewURLChanged(key, url.url());
        });

        QObject::connect(view, &QWebEngineView::urlChanged, [this, key](const QUrl &url) {
            emit controlInterface.onWebViewHostnameChanged(key, url.host());
        });

        QObject::connect(view, &QWebEngineView::titleChanged, [this, key](const QString &title) {
            emit controlInterface.onWebViewTitleChanged(key, title);
        });

        QObject::connect(view, &QWebEngineView::loadProgress, [this, key](int progress) {
            emit controlInterface.onWebViewLoadProgressChanged(key, progress);
        });

        //QObject::connect(page, &WebEnginePage::onCertificateInvalid, [this, key](const QUrl &url) {
        //    emit controlInterface.onCertificateInvalid(key, url);
        //});

        return key;
    });

    QObject::connect(&controlInterface, &ControlInterface::onWebViewForwardHistoryRequest, [this](int key){
        QWebEngineView *view = lookupWebView(key);
        if (view) {
            QStringList pages;

            auto items = view->page()->history()->forwardItems(1000);
            foreach (auto item, items) {
                pages << item.url().toString();
            }

            emit controlInterface.onWebViewForwardHistoryRequested(key, pages);
        }
    });

    QObject::connect(&controlInterface, &ControlInterface::onWebViewBackwardHistoryRequest, [this](int key){
        QWebEngineView *view = lookupWebView(key);
        if (view) {
            QStringList pages;

            auto items = view->page()->history()->backItems(1000);
            foreach (auto item, items) {
                pages << item.url().toString();
            }

            emit controlInterface.onWebViewBackwardHistoryRequested(key, pages);
        }
    });

    QObject::connect(&controlInterface, &ControlInterface::onDestroyWebViewRequest, [this](int key) {
        QWebEngineView *view = lookupWebView(key);
        if (view) {
            view->setVisible(false);
            views.remove(key);
            delete view;
        }
    });

    QObject::connect(&controlInterface, &ControlInterface::onWebViewURLChangeRequested, [this](int key, const QString &url) {
        QWebEngineView *view = lookupWebView(key);
        if (view)
            view->setUrl(QUrl(url));
    });

    QObject::connect(&controlInterface, &ControlInterface::onWebViewGeometryChangeRequest, [this](int key, int x, int y, int w, int h) {
        QWebEngineView *view = lookupWebView(key);
        if (view)
            view->setGeometry(x, y, w, h);
    });

    QObject::connect(&controlInterface, &ControlInterface::onWebViewVisibleChangeRequest, [this](int key, bool value) {
        QWebEngineView *view = lookupWebView(key);
        if (view)
            view->setVisible(value);
    });

    QObject::connect(&controlInterface, &ControlInterface::onWebViewTransparentBackgroundChangeRequest, [this](int key, bool value) {

        QWebEngineView *view = lookupWebView(key);
        if (view) {
            if (value) {
                view->setAutoFillBackground(false);
                view->page()->setBackgroundColor(Qt::transparent);
            } else {
                view->setAutoFillBackground(true);
                view->page()->setBackgroundColor(Qt::white);
            }
        }
    });

    QObject::connect(&controlInterface, &ControlInterface::onWebViewNextRequest, [this](int key){
        QWebEngineView *view = lookupWebView(key);
        if (!view) return;

        WebEnginePage *page = static_cast<WebEnginePage*>(view->page());
        if (!page)
            return;

        page->next();
    });

    QObject::connect(&controlInterface, &ControlInterface::onWebViewPrevRequest, [this](int key){
        QWebEngineView *view = lookupWebView(key);
        if (!view)
            return;

        WebEnginePage *page = static_cast<WebEnginePage*>(view->page());
        if (!page)
            return;

        page->prev();
    });

    QObject::connect(&controlInterface, &ControlInterface::onWebViewStackUnderRequest, [this](int topKey, int underKey) {

        auto wTop = lookupWebView(topKey);
        auto wUnder = lookupWebView(underKey);
        if (wTop && wUnder)
            wTop->stackUnder(wUnder);
    });

    QObject::connect(&controlInterface, &ControlInterface::onWebViewStackOnTopRequest, [this](int key) {

        auto w = lookupWebView(key);
        if (w)
            w->raise();
    });

    QObject::connect(&controlInterface, &ControlInterface::onKeyboardShowRequest, [this](void) {
        onKeyboardActiveChanged(true);
    });

    QObject::connect(&controlInterface, &ControlInterface::onKeyboardHideRequest, [this](void) {
        onKeyboardActiveChanged(false);
    });

    QObject::connect(&controlInterface, &ControlInterface::onSetKeyboardEnabledRequested, [this](bool enabled){

        if (!enabled)
            quickWidget->setVisible(false);

        this->keyboardEnabled = enabled;
    });

    controlChannel.registerObject(QStringLiteral("main"), &controlInterface);
}

int MainWindow::nextKey()
{
    static int nextKey = 0;
    return nextKey++;
}


QWebEngineView *MainWindow::addWebView(int key)
{
    QWebEngineView *view = new QWebEngineView(browserWidget);
    view->setAutoFillBackground(true);
    view->setContextMenuPolicy(Qt::NoContextMenu);

    WebEnginePage *page = new WebEnginePage(QWebEngineProfile::defaultProfile(), view);
    page->setBackgroundColor(Qt::transparent);
    view->setPage(page);

    views.insert(key, view);

    return view;
}

QWebEngineView *MainWindow::lookupWebView(int key)
{
    if (views.contains(key))
        return views.value(key);

    return Q_NULLPTR;
}

QWebEngineView *MainWindow::lookupVisibleWebView()
{
    for (int i=0; i<views.size(); i++)
        if (views[i]->isVisible())
            return views[i];

    return Q_NULLPTR;
}
