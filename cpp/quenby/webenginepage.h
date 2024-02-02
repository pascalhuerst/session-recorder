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

#pragma once

#include <QWebEnginePage>
#include <QSet>
#include <QUrl>

class WebEnginePage : public QWebEnginePage
{
    Q_OBJECT

public:
    WebEnginePage(QWebEngineProfile *profile, QObject *parent = Q_NULLPTR);
    void prev(void);
    void next(void);

//signals:
//    void onCertificateInvalid(const QUrl &url);
//
//protected:
//    virtual bool certificateError(const QWebEngineCertificateError &certificateError);

private:
    QSet<QUrl> m_urlWithCertError;

};
