include $(TOPDIR)/rules.mk

PKG_NAME:=heishamon
PKG_VERSION:=2.1.0-lr
PKG_RELEASE:=1

PKG_SOURCE:=$(PKG_NAME)-$(PKG_VERSION).tar.gz
PKG_SOURCE_URL:=https://codeload.github.com/rondoval/GoHeishaMon/tar.gz/v$(PKG_VERSION)?
PKG_HASH:=294300a8c182c3f8e1f537ad2feebb6d0651f61330f33504bdc502f48992bf84

PKG_MAINTAINER:=Lukasz Winczura <rondoval@gmail.com>
PKG_LICENSE:=MIT
PKG_LICENSE_FILES:=LICENSE

PKG_BUILD_DEPENDS:=golang/host
PKG_BUILD_PARALLEL:=1
PKG_USE_MIPS16:=0

GO_PKG:=github.com/rondoval/GoHeishaMon


include $(INCLUDE_DIR)/package.mk
include ../../lang/golang/golang-package.mk

define Package/heishamon
  SECTION:=utils
  CATEGORY:=Utilities
  TITLE:=Panasonic Aquarea to MQTT gateway
  URL:=https://github.com/rondoval/GoHeishaMon
  DEPENDS:=$(GO_ARCH_DEPENDS)
endef

define Package/heishamon/description
  This is a Panasonic Aquarea H-series heat pump to MQTT gateway.
  Uses a serial connection to the heat pump's CN-CNT connector.
endef

define Package/heishamon/conffiles
/etc/heishamon/config.yaml
/etc/heishamon/optionalpcb.raw
endef

define Package/heishamon/install
	$(call GoPackage/Package/Install/Bin,$(PKG_INSTALL_DIR))

	$(INSTALL_DIR) $(1)/usr/bin
	$(INSTALL_BIN) $(PKG_INSTALL_DIR)/usr/bin/* $(1)/usr/bin/

	$(INSTALL_DIR) $(1)/etc/heishamon
	$(INSTALL_CONF) $(PKG_BUILD_DIR)/config.yaml.example $(1)/etc/heishamon/config.yaml
	$(INSTALL_CONF) $(PKG_BUILD_DIR)/topics.yaml $(1)/etc/heishamon/topics.yaml

	$(INSTALL_DIR) $(1)/etc/init.d
	$(INSTALL_BIN) $(PKG_BUILD_DIR)/heishamon.init $(1)/etc/init.d/heishamon
endef

$(eval $(call GoBinPackage,heishamon))
$(eval $(call BuildPackage,heishamon))
