include $(TOPDIR)/rules.mk

PKG_NAME:=heishamon
PKG_VERSION:=3.3.2
PKG_RELEASE:=1

PKG_MAINTAINER:=Lukasz Winczura <rondoval@gmail.com>
PKG_LICENSE:=MIT
PKG_LICENSE_FILES:=LICENSE

PKG_BUILD_DIR:=$(BUILD_DIR)/heishamon-$(PKG_VERSION)
PKG_BUILD_DEPENDS:=golang/host
PKG_BUILD_PARALLEL:=1
PKG_USE_MIPS16:=0
PKG_BUILD_FLAGS:=no-mips16

GO_PKG:=github.com/rondoval/GoHeishaMon


include $(INCLUDE_DIR)/package.mk
include $(TOPDIR)/feeds/packages/lang/golang/golang-package.mk

define Package/heishamon
  SECTION:=utils
  CATEGORY:=Utilities
  TITLE:=Panasonic Aquarea to MQTT gateway
  URL:=https://github.com/rondoval/GoHeishaMon
  DEPENDS:=$(GO_ARCH_DEPENDS)
endef

define Package/heishamon/description
 This is a Panasonic Aquarea H-series heat pump to MQTT gateway.
endef

define Package/heishamon/conffiles
/etc/heishamon/config.yaml
/etc/heishamon/optionalpcb.yaml
endef

define Package/heishamon/install
	$(call GoPackage/Package/Install/Bin,$(PKG_INSTALL_DIR))

	$(INSTALL_DIR) $(1)/usr/bin
	$(INSTALL_BIN) $(PKG_INSTALL_DIR)/usr/bin/GoHeishaMon $(1)/usr/bin/heishamon

	$(INSTALL_DIR) $(1)/etc/heishamon
	$(INSTALL_CONF) ./files/config.yaml.example $(1)/etc/heishamon/config.yaml
	$(INSTALL_CONF) ./files/topics.yaml $(1)/etc/heishamon/topics.yaml
	$(INSTALL_CONF) ./files/topicsOptionalPCB.yaml $(1)/etc/heishamon/topicsOptionalPCB.yaml

	$(INSTALL_DIR) $(1)/etc/init.d
	$(INSTALL_BIN) ./files/heishamon.init $(1)/etc/init.d/heishamon
endef

$(eval $(call GoBinPackage,heishamon))
$(eval $(call BuildPackage,heishamon))
