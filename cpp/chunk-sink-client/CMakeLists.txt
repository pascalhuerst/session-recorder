cmake_minimum_required(VERSION 3.10...3.31)

project(chunk-sink-client C CXX)

include(FindPkgConfig)
include(CMakeFindDependencyMacro)

function(addLib name)
  pkg_check_modules(${name} REQUIRED ${name})
  include_directories(${${name}_INCLUDE_DIRS})
  link_directories(${${name}_LIBRARY_DIRS})
endfunction(addLib)

function(linkLib name target)
  pkg_check_modules(${name} REQUIRED ${name})
  target_link_libraries(${target} ${${name}_LIBRARIES})
endfunction(linkLib)

file(GLOB GRPC_SOURCES "${CMAKE_CURRENT_SOURCE_DIR}/../../protocols/cpp/*.cc")

set(LIBS alsa avahi-client)
set(SOURCES
  ${GRPC_SOURCES}

  chunk-sink-client.cpp
  AlsaAudioInput.cpp
  publishZeroConf/publishAvahi.cpp
  AudioStreamManager.cpp
  InputKey.cpp
  Led.cpp
  ServiceTracker.cpp
)
set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -std=c++20 -Wno-unused-parameter -Wall -O2")

foreach(lib ${LIBS})
  addLib(${lib})
endforeach(lib)

# Use pkg-config for gRPC and protobuf (more portable across distros)
pkg_check_modules(GRPC REQUIRED grpc++)
pkg_check_modules(PROTOBUF REQUIRED protobuf)

find_package(Threads REQUIRED)
find_package(Boost 1.74 COMPONENTS program_options REQUIRED)

include_directories("${CMAKE_CURRENT_BINARY_DIR}")
include_directories("${CMAKE_CURRENT_SOURCE_DIR}/../../protocols/cpp")
include_directories(${GRPC_INCLUDE_DIRS})
include_directories(${PROTOBUF_INCLUDE_DIRS})

add_executable(${PROJECT_NAME} ${SOURCES})

foreach(lib ${LIBS})
  linkLib(${lib} ${PROJECT_NAME})
endforeach(lib)

message(STATUS "gRPC libraries: ${GRPC_LIBRARIES}")
message(STATUS "Protobuf libraries: ${PROTOBUF_LIBRARIES}")

target_link_libraries(${PROJECT_NAME}
  ${CMAKE_THREAD_LIBS_INIT}
  ${Boost_LIBRARIES}
  ${PROTOBUF_LIBRARIES}
  ${GRPC_LIBRARIES}
)

install(TARGETS ${PROJECT_NAME} RUNTIME DESTINATION sbin COMPONENT runtime)

# Install systemd service file
install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/packaging/chunk-sink-client.service
        DESTINATION lib/systemd/system
        COMPONENT runtime)

# Install default configuration
install(FILES ${CMAKE_CURRENT_SOURCE_DIR}/packaging/chunk-sink-client.conf
        DESTINATION etc/default
        RENAME chunk-sink-client
        COMPONENT runtime)

# =============================================================================
# CPack Configuration - Multi-distro packaging
# =============================================================================
set(CPACK_PACKAGE_NAME "chunk-sink-client")
set(CPACK_PACKAGE_VENDOR "Session Recorder Project")
set(CPACK_PACKAGE_DESCRIPTION_SUMMARY "Audio chunk streaming client for session recording")
set(CPACK_PACKAGE_DESCRIPTION "Captures audio from ALSA devices and streams chunks to a backend server via gRPC. Supports mDNS service discovery.")
set(CPACK_PACKAGE_HOMEPAGE_URL "https://github.com/pascalhuerst/session-recorder")
set(CPACK_PACKAGE_VERSION_MAJOR "1")
set(CPACK_PACKAGE_VERSION_MINOR "0")
set(CPACK_PACKAGE_VERSION_PATCH "0")
set(CPACK_PACKAGE_VERSION "${CPACK_PACKAGE_VERSION_MAJOR}.${CPACK_PACKAGE_VERSION_MINOR}.${CPACK_PACKAGE_VERSION_PATCH}")
set(CPACK_PACKAGE_CONTACT "Pascal Huerst <pascal.huerst@gmail.com>")
set(CPACK_PACKAGE_MAINTAINER "Pascal Huerst <pascal.huerst@gmail.com>")
set(CPACK_RESOURCE_FILE_LICENSE "${CMAKE_CURRENT_SOURCE_DIR}/LICENSE")

# Package file naming
set(CPACK_PACKAGE_FILE_NAME "${CPACK_PACKAGE_NAME}-${CPACK_PACKAGE_VERSION}-${CMAKE_SYSTEM_PROCESSOR}")

# -----------------------------------------------------------------------------
# DEB Package Configuration (Debian, Ubuntu, Raspberry Pi OS)
# -----------------------------------------------------------------------------
set(CPACK_DEBIAN_PACKAGE_MAINTAINER "${CPACK_PACKAGE_MAINTAINER}")
set(CPACK_DEBIAN_PACKAGE_SECTION "sound")
set(CPACK_DEBIAN_PACKAGE_PRIORITY "optional")
set(CPACK_DEBIAN_PACKAGE_DEPENDS
    "libasound2, libavahi-client3, libavahi-common3, libboost-program-options1.74.0 | libboost-program-options1.81.0 | libboost-program-options1.83.0, libgrpc++1.51 | libgrpc++1.58 | libgrpc++1, libprotobuf32 | libprotobuf23 | libprotobuf35")
set(CPACK_DEBIAN_PACKAGE_RECOMMENDS "avahi-daemon")
set(CPACK_DEBIAN_PACKAGE_CONTROL_EXTRA
    "${CMAKE_CURRENT_SOURCE_DIR}/packaging/postinst;${CMAKE_CURRENT_SOURCE_DIR}/packaging/prerm")
set(CPACK_DEBIAN_FILE_NAME DEB-DEFAULT)

# -----------------------------------------------------------------------------
# RPM Package Configuration (Fedora, RHEL, openSUSE)
# -----------------------------------------------------------------------------
set(CPACK_RPM_PACKAGE_LICENSE "GPLv3")
set(CPACK_RPM_PACKAGE_GROUP "Applications/Multimedia")
set(CPACK_RPM_PACKAGE_REQUIRES
    "alsa-lib, avahi-libs, boost-program-options, grpc-cpp, protobuf")
set(CPACK_RPM_PACKAGE_SUGGESTS "avahi")
set(CPACK_RPM_POST_INSTALL_SCRIPT_FILE "${CMAKE_CURRENT_SOURCE_DIR}/packaging/postinst.rpm")
set(CPACK_RPM_PRE_UNINSTALL_SCRIPT_FILE "${CMAKE_CURRENT_SOURCE_DIR}/packaging/prerm.rpm")
set(CPACK_RPM_FILE_NAME RPM-DEFAULT)
set(CPACK_RPM_PACKAGE_AUTOREQ OFF)

# -----------------------------------------------------------------------------
# TGZ Package Configuration (Universal tarball)
# -----------------------------------------------------------------------------
set(CPACK_ARCHIVE_COMPONENT_INSTALL OFF)

# Generator selection
set(CPACK_GENERATOR "DEB;RPM;TGZ")

include(CPack)
