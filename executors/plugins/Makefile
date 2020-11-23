# make build
# make clean
# make dist

DIRS 			:= `ls -d */ | cut -f1 -d'/'|grep -v dist`
ALL_PLUGINS 	:= $(if ${TARGET_NAME},${TARGET_NAME}, $(filter-out $(TARGET_DIST), $(shell for plugin in $(DIRS); do echo $(basename "$$plugin"); done)))
TARGET_PLUGINS 	:= $(foreach PLUGIN, $(ALL_PLUGINS), dist/lib/$(PLUGIN).so)
TARGET_DIST 	:= ./dist

define get_plugin_name
$(strip $(patsubst dist/lib/%.so, %,$(shell echo $(1))))
endef

$(TARGET_PLUGINS):
	@mkdir -p dist
	$(info building $(call get_plugin_name,$@))
	@P=$(call get_plugin_name,$@); \
	$(MAKE) build -C $$P

build: $(TARGET_PLUGINS)

.PHONY: dist
dist:
	@mkdir -p dist/lib
	@for PLU in $(ALL_PLUGINS); do \
		mv $$PLU/dist/lib/* dist/lib; \
	done;

.PHONY: clean
clean:
	@rm -rf $(TARGET_DIST)
	@for P in $(ALL_PLUGINS); do \
		$(MAKE) clean -C $$P; \
	done;
