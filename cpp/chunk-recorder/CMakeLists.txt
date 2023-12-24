cmake_minimum_required(VERSION 3.5.1)

project(chunk-recorder C CXX)

include(FindPkgConfig)

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

set(LIBS alsa avahi-client avahi-core)
set(SOURCES 
  ${GRPC_SOURCES}
  
  chunk-recorder.cpp
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

#find_package(protobuf REQUIRED)
find_package(gRPC CONFIG REQUIRED)
find_package(Threads REQUIRED)
find_package(Boost 1.81 COMPONENTS program_options REQUIRED)

include_directories("${CMAKE_CURRENT_BINARY_DIR}")
include_directories("${CMAKE_CURRENT_SOURCE_DIR}/../../protocols/cpp")

add_executable(${PROJECT_NAME} ${SOURCES})

foreach(lib ${LIBS})
  linkLib(${lib} ${PROJECT_NAME})
endforeach(lib)

message(STATUS "gRPC libraries: ${gRPC_LIBRARIES}")
message(STATUS "Protobuf libraries: ${Protobuf_LIBRARIES}")

target_link_libraries(${PROJECT_NAME}
  libstdc++fs.a
  ${CMAKE_THREAD_LIBS_INIT}
  ${Boost_LIBRARIES}
  protobuf::libprotobuf
  gRPC::grpc
  gRPC::grpc++
)

install(TARGETS ${PROJECT_NAME} RUNTIME DESTINATION /usr/sbin COMPONENT runtime)
