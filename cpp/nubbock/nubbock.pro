QT += gui gui-private core-private waylandcompositor waylandcompositor-private

LIBS += -L ../../lib

HEADERS += \
    compositor.h \
    window.h \
    socketserver.h

SOURCES += main.cpp \
    compositor.cpp \
    window.cpp \
    socketserver.cpp
