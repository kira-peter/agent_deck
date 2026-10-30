package i18n

// Texts for the model picker.
var (
	PickTitle     = Text{En: "Choose model", Zh: "选择模型"}
	PickCurrent   = Text{En: "Current", Zh: "当前"}
	PickNotInList = Text{En: "(not in the list)", Zh: "(不在列表中)"}
	PickNotSet    = Text{En: "(not set)", Zh: "(未设置)"}
	PickVendor    = Text{En: "Vendor", Zh: "厂家"}
	PickModels    = Text{En: "Models", Zh: "模型"}
	PickCustom    = Text{En: "Custom…", Zh: "自定义…"}
	PickCustomTip = Text{
		En: "Type any model id that is not in the list, then press Enter.",
		Zh: "输入任意不在列表里的模型 ID，按 Enter 确认。",
	}
	PickInUse = Text{En: "in use", Zh: "使用中"}

	PickHintVendor = Text{
		En: "↑↓: Move  →: Enter list  ESC: Cancel",
		Zh: "↑↓: 移动  →: 进入列表  ESC: 取消",
	}
	PickHintModels = Text{
		En: "↑↓: Move  ←: Back to vendors  Enter: Select  ESC: Cancel",
		Zh: "↑↓: 移动  ←: 定位到厂家  Enter: 选择  ESC: 取消",
	}
	PickHintCustom = Text{
		En: "Type a model id  Enter: Select  ↑↓: Change vendor  ESC: Cancel",
		Zh: "直接输入模型 ID  Enter: 确认  ↑↓: 切换厂家  ESC: 取消",
	}
	PickEmptyCustom = Text{En: "Model id cannot be empty", Zh: "模型 ID 不能为空"}
)
