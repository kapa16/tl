
#Область ОбработчикиСобытийФормы

&НаСервере
Процедура ПриСозданииНаСервере(Отказ, СтандартнаяОбработка)
	
	Заголовок = Нстр("ru = 'Настройка отчета ""Статистика""'; en = 'Setting up the ""Statistics"" report'");
	
	Если Параметры.Свойство("АвтоТест") Тогда // Возврат при получении формы для анализа.
		Автотест = Истина;
		Возврат;
	КонецЕсли;
	
	ЗаполнитьЗаголовкиФлагов();
	
КонецПроцедуры

&НаКлиенте
Процедура ПриОткрытии(Отказ)
	
	Если Автотест Тогда // Возврат при получении формы для анализа.
		Возврат;
	КонецЕсли;
	
	Если ВладелецФормы = Неопределено Тогда
		Отказ = Истина;
		ПоказатьПредупреждение(Неопределено, НСтр("en='Immediate opening for this object is prohibited!';ru='Непосредственное открытие для данного объекта запрещено!'"));
		Возврат;
	КонецЕсли;
	
	ВосстановитьНастройки();
	
КонецПроцедуры

#КонецОбласти

#Область ОбработчикиКомандФормы

&НаКлиенте
Процедура СохранитьНастройкиОтчета(Команда)
	СохранитьНастройки();
КонецПроцедуры

#КонецОбласти

#Область СлужебныеПроцедурыИФункции

&НаСервере
Процедура ВосстановитьНастройки()
	
	УстановитьПривилегированныйРежим(Истина);
	СтруктураНастроек = ХранилищеНастроекДанныхФорм.Загрузить(
		"уатСтатистикаOmnicomm.ФормаНастроекОтчета", 
		"ФормаНастроекОтчета",,Пользователи.АвторизованныйПользователь());
	
	Если ТипЗнч(СтруктураНастроек) <> Тип("Структура") Тогда
		СтруктураНастроек = Новый Структура();
		СтруктураНастроек.Вставить("movementAndWorkingParams", Истина);
		СтруктураНастроек.Вставить("mileage", Истина);
		СтруктураНастроек.Вставить("mileageAverage", Истина);
		СтруктураНастроек.Вставить("mileageSpeeding", Истина);
		СтруктураНастроек.Вставить("speedAverage", Истина);
		СтруктураНастроек.Вставить("speedMax", Истина);
		СтруктураНастроек.Вставить("movementTimeTOtal", Истина);
		СтруктураНастроек.Вставить("movementTime", Истина);
		СтруктураНастроек.Вставить("engineOperationTime", Истина);
		СтруктураНастроек.Вставить("engineOperationTimeInMovement", Истина);
		СтруктураНастроек.Вставить("engineOperationTimeWithoutMovement", Истина);
		СтруктураНастроек.Вставить("engineIdlingTime", Истина);
		СтруктураНастроек.Вставить("engineOperationTimeNormalSpeed", Истина);
		СтруктураНастроек.Вставить("engineOperationTimeMaxSpeed", Истина);
		СтруктураНастроек.Вставить("engineOffTime", Истина);
		СтруктураНастроек.Вставить("odometerInitial", Истина);
		СтруктураНастроек.Вставить("odometerFinal", Истина);
		
		СтруктураНастроек.Вставить("fuelParams", Истина);	
		СтруктураНастроек.Вставить("fuelVolumeInitial", Истина);
		СтруктураНастроек.Вставить("fuelVolumeFinal", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionActual", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionActualAverage", Истина);
		СтруктураНастроек.Вставить("fuellingsVolume", Истина);
		СтруктураНастроек.Вставить("fillsVolume", Истина);
		СтруктураНастроек.Вставить("drainingsVolume", Истина);
		СтруктураНастроек.Вставить("fuelDispensingVolume", Истина);
		СтруктураНастроек.Вставить("possibleDrainOrExcess", Истина);
		СтруктураНастроек.Вставить("fuelVolumeMin", Истина);
		СтруктураНастроек.Вставить("fuelVolumeMax", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionActualPer100km", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionActualInMovementPer100km", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionActualInMovement", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionActualWithoutMovement", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionRatedPer100km", Истина);
		СтруктураНастроек.Вставить("fuelVolumeCalculatedByFuelConsumptionRatedPer100km", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionDeviationInPercentPer100km", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionDeviationPer100km", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionActualPer1HourEngineOperation", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionActualPer1HourEngineOperationWithoutMovement", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionRatedPer1HourEngineOperation", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionRatedByPeriod", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionDeviationInPercentPer1HourEngineOperation", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionDeviationPer1HourEngineOperation", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionActualInMovementIdleEngineSpeed", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionActualWithoutMovementIdleEngineSpeed", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionActualInMovementNormalEngineSpeed", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionActualWithoutMovementNormalEngineSpeed", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionActualInMovementMaxEngineSpeed", Истина);
		СтруктураНастроек.Вставить("fuelConsumptionActualWithoutMovementMaxEngineSpeed", Истина);
		СтруктураНастроек.Вставить("engineIdleFuelConsumptionDeviationFromRatedPer1Hour", Истина);
		
	КонецЕсли;
	
	СтруктураНастроек.Свойство("movementAndWorkingParams", movementAndWorkingParams);
	
	СтруктураНастроек.Свойство("mileage", mileage);
	СтруктураНастроек.Свойство("mileageAverage", mileageAverage);
	СтруктураНастроек.Свойство("mileageSpeeding", mileageSpeeding);
	СтруктураНастроек.Свойство("speedAverage", speedAverage);
	СтруктураНастроек.Свойство("speedMax", speedMax);
	СтруктураНастроек.Свойство("movementTimeTOtal", movementTimeTOtal);
	СтруктураНастроек.Свойство("movementTime", movementTime);
	СтруктураНастроек.Свойство("engineOperationTime", engineOperationTime);
	СтруктураНастроек.Свойство("engineOperationTimeInMovement", engineOperationTimeInMovement);
	СтруктураНастроек.Свойство("engineOperationTimeWithoutMovement", engineOperationTimeWithoutMovement);
	СтруктураНастроек.Свойство("engineIdlingTime", engineIdlingTime);
	СтруктураНастроек.Свойство("engineOperationTimeNormalSpeed", engineOperationTimeNormalSpeed);
	СтруктураНастроек.Свойство("engineOperationTimeMaxSpeed", engineOperationTimeMaxSpeed);
	СтруктураНастроек.Свойство("engineOffTime", engineOffTime);
	СтруктураНастроек.Свойство("odometerInitial", odometerInitial);
	СтруктураНастроек.Свойство("odometerFinal", odometerFinal);

	СтруктураНастроек.Свойство("fuelParams", fuelParams);
	
	СтруктураНастроек.Свойство("fuelVolumeInitial", fuelVolumeInitial);
	СтруктураНастроек.Свойство("fuelVolumeFinal", fuelVolumeFinal);
	СтруктураНастроек.Свойство("fuelConsumptionActual", fuelConsumptionActual);
	СтруктураНастроек.Свойство("fuelConsumptionActualAverage", fuelConsumptionActualAverage);
	СтруктураНастроек.Свойство("fuellingsVolume", fuellingsVolume);
	СтруктураНастроек.Свойство("fillsVolume", fillsVolume);
	СтруктураНастроек.Свойство("drainingsVolume", drainingsVolume);
	СтруктураНастроек.Свойство("fuelDispensingVolume", fuelDispensingVolume);
	СтруктураНастроек.Свойство("possibleDrainOrExcess", possibleDrainOrExcess);
	СтруктураНастроек.Свойство("fuelVolumeMin", fuelVolumeMin);
	СтруктураНастроек.Свойство("fuelVolumeMax", fuelVolumeMax);
	СтруктураНастроек.Свойство("fuelConsumptionActualPer100km", fuelConsumptionActualPer100km);
	СтруктураНастроек.Свойство("fuelConsumptionActualInMovementPer100km", fuelConsumptionActualInMovementPer100km);
	СтруктураНастроек.Свойство("fuelConsumptionActualInMovement", fuelConsumptionActualInMovement);
	СтруктураНастроек.Свойство("fuelConsumptionActualWithoutMovement", fuelConsumptionActualWithoutMovement);
	СтруктураНастроек.Свойство("fuelConsumptionRatedPer100km", fuelConsumptionRatedPer100km);
	СтруктураНастроек.Свойство("fuelVolumeCalculatedByFuelConsumptionRatedPer100km", fuelVolumeCalculatedByFuelConsumptionRatedPer100km);
	СтруктураНастроек.Свойство("fuelConsumptionDeviationInPercentPer100km", fuelConsumptionDeviationInPercentPer100km);
	СтруктураНастроек.Свойство("fuelConsumptionDeviationPer100km", fuelConsumptionDeviationPer100km);
	СтруктураНастроек.Свойство("fuelConsumptionActualPer1HourEngineOperation", fuelConsumptionActualPer1HourEngineOperation);
	СтруктураНастроек.Свойство("fuelConsumptionActualPer1HourEngineOperationWithoutMovement", fuelConsumptionActualPer1HourEngineOperationWithoutMovement);
	СтруктураНастроек.Свойство("fuelConsumptionRatedPer1HourEngineOperation", fuelConsumptionRatedPer1HourEngineOperation);
	СтруктураНастроек.Свойство("fuelConsumptionRatedByPeriod", fuelConsumptionRatedByPeriod);
	СтруктураНастроек.Свойство("fuelConsumptionDeviationInPercentPer1HourEngineOperation", fuelConsumptionDeviationInPercentPer1HourEngineOperation);
	СтруктураНастроек.Свойство("fuelConsumptionDeviationPer1HourEngineOperation", fuelConsumptionDeviationPer1HourEngineOperation);
	СтруктураНастроек.Свойство("fuelConsumptionActualInMovementIdleEngineSpeed", fuelConsumptionActualInMovementIdleEngineSpeed);
	СтруктураНастроек.Свойство("fuelConsumptionActualWithoutMovementIdleEngineSpeed", fuelConsumptionActualWithoutMovementIdleEngineSpeed);
	СтруктураНастроек.Свойство("fuelConsumptionActualInMovementNormalEngineSpeed", fuelConsumptionActualInMovementNormalEngineSpeed);
	СтруктураНастроек.Свойство("fuelConsumptionActualWithoutMovementNormalEngineSpeed", fuelConsumptionActualWithoutMovementNormalEngineSpeed);
	СтруктураНастроек.Свойство("fuelConsumptionActualInMovementMaxEngineSpeed", fuelConsumptionActualInMovementMaxEngineSpeed);
	СтруктураНастроек.Свойство("fuelConsumptionActualWithoutMovementMaxEngineSpeed", fuelConsumptionActualWithoutMovementMaxEngineSpeed);
	СтруктураНастроек.Свойство("engineIdleFuelConsumptionDeviationFromRatedPer1Hour", engineIdleFuelConsumptionDeviationFromRatedPer1Hour);
	

КонецПроцедуры

&НаСервере
Процедура СохранитьНастройки()
	
	УстановитьПривилегированныйРежим(Истина);
	
	СтруктураНастроек = Новый Структура();
	СтруктураНастроек.Вставить("movementAndWorkingParams", movementAndWorkingParams);
	
	СтруктураНастроек.Вставить("mileage", mileage);
	СтруктураНастроек.Вставить("mileageAverage", mileageAverage);
	СтруктураНастроек.Вставить("mileageSpeeding", mileageSpeeding);
	СтруктураНастроек.Вставить("speedAverage", speedAverage);
	СтруктураНастроек.Вставить("speedMax", speedMax);
	СтруктураНастроек.Вставить("movementTimeTOtal", movementTimeTOtal);
	СтруктураНастроек.Вставить("movementTime", movementTime);
	СтруктураНастроек.Вставить("engineOperationTime", engineOperationTime);
	СтруктураНастроек.Вставить("engineOperationTimeInMovement", engineOperationTimeInMovement);
	СтруктураНастроек.Вставить("engineOperationTimeWithoutMovement", engineOperationTimeWithoutMovement);
	СтруктураНастроек.Вставить("engineIdlingTime", engineIdlingTime);
	СтруктураНастроек.Вставить("engineOperationTimeNormalSpeed", engineOperationTimeNormalSpeed);
	СтруктураНастроек.Вставить("engineOperationTimeMaxSpeed", engineOperationTimeMaxSpeed);
	СтруктураНастроек.Вставить("engineOffTime", engineOffTime);
	СтруктураНастроек.Вставить("odometerInitial", odometerInitial);
	СтруктураНастроек.Вставить("odometerFinal", odometerFinal);
	
	СтруктураНастроек.Вставить("fuelParams", fuelParams);
	
	СтруктураНастроек.Вставить("fuelVolumeInitial", fuelVolumeInitial);
	СтруктураНастроек.Вставить("fuelVolumeFinal", fuelVolumeFinal);
	СтруктураНастроек.Вставить("fuelConsumptionActual", fuelConsumptionActual);
	СтруктураНастроек.Вставить("fuelConsumptionActualAverage", fuelConsumptionActualAverage);
	СтруктураНастроек.Вставить("fuellingsVolume", fuellingsVolume);
	СтруктураНастроек.Вставить("fillsVolume", fillsVolume);
	СтруктураНастроек.Вставить("drainingsVolume", drainingsVolume);
	СтруктураНастроек.Вставить("fuelDispensingVolume", fuelDispensingVolume);
	СтруктураНастроек.Вставить("possibleDrainOrExcess", possibleDrainOrExcess);
	СтруктураНастроек.Вставить("fuelVolumeMin", fuelVolumeMin);
	СтруктураНастроек.Вставить("fuelVolumeMax", fuelVolumeMax);
	СтруктураНастроек.Вставить("fuelConsumptionActualPer100km", fuelConsumptionActualPer100km);
	СтруктураНастроек.Вставить("fuelConsumptionActualInMovementPer100km", fuelConsumptionActualInMovementPer100km);
	СтруктураНастроек.Вставить("fuelConsumptionActualInMovement", fuelConsumptionActualInMovement);
	СтруктураНастроек.Вставить("fuelConsumptionActualWithoutMovement", fuelConsumptionActualWithoutMovement);
	СтруктураНастроек.Вставить("fuelConsumptionRatedPer100km", fuelConsumptionRatedPer100km);
	СтруктураНастроек.Вставить("fuelVolumeCalculatedByFuelConsumptionRatedPer100km", fuelVolumeCalculatedByFuelConsumptionRatedPer100km);
	СтруктураНастроек.Вставить("fuelConsumptionDeviationInPercentPer100km", fuelConsumptionDeviationInPercentPer100km);
	СтруктураНастроек.Вставить("fuelConsumptionDeviationPer100km", fuelConsumptionDeviationPer100km);
	СтруктураНастроек.Вставить("fuelConsumptionActualPer1HourEngineOperation", fuelConsumptionActualPer1HourEngineOperation);
	СтруктураНастроек.Вставить("fuelConsumptionActualPer1HourEngineOperationWithoutMovement", fuelConsumptionActualPer1HourEngineOperationWithoutMovement);
	СтруктураНастроек.Вставить("fuelConsumptionRatedPer1HourEngineOperation", fuelConsumptionRatedPer1HourEngineOperation);
	СтруктураНастроек.Вставить("fuelConsumptionRatedByPeriod", fuelConsumptionRatedByPeriod);
	СтруктураНастроек.Вставить("fuelConsumptionDeviationInPercentPer1HourEngineOperation", fuelConsumptionDeviationInPercentPer1HourEngineOperation);
	СтруктураНастроек.Вставить("fuelConsumptionDeviationPer1HourEngineOperation", fuelConsumptionDeviationPer1HourEngineOperation);
	СтруктураНастроек.Вставить("fuelConsumptionActualInMovementIdleEngineSpeed", fuelConsumptionActualInMovementIdleEngineSpeed);
	СтруктураНастроек.Вставить("fuelConsumptionActualWithoutMovementIdleEngineSpeed", fuelConsumptionActualWithoutMovementIdleEngineSpeed);
	СтруктураНастроек.Вставить("fuelConsumptionActualInMovementNormalEngineSpeed", fuelConsumptionActualInMovementNormalEngineSpeed);
	СтруктураНастроек.Вставить("fuelConsumptionActualWithoutMovementNormalEngineSpeed", fuelConsumptionActualWithoutMovementNormalEngineSpeed);
	СтруктураНастроек.Вставить("fuelConsumptionActualInMovementMaxEngineSpeed", fuelConsumptionActualInMovementMaxEngineSpeed);
	СтруктураНастроек.Вставить("fuelConsumptionActualWithoutMovementMaxEngineSpeed", fuelConsumptionActualWithoutMovementMaxEngineSpeed);
	СтруктураНастроек.Вставить("engineIdleFuelConsumptionDeviationFromRatedPer1Hour", engineIdleFuelConsumptionDeviationFromRatedPer1Hour);
	
	ХранилищеНастроекДанныхФорм.Сохранить(
		"уатСтатистикаOmnicomm.ФормаНастроекОтчета", 
		"ФормаНастроекОтчета",
		СтруктураНастроек,,Пользователи.АвторизованныйПользователь());
		
КонецПроцедуры

&НаСервере
Процедура ЗаполнитьЗаголовкиФлагов()
	ИмяМакета = "НастройкиПоУмолчанию";
	МакетНастройки	 = УправлениеПечатью.МакетПечатнойФормы("ОбщийМакет.уатНастройкиОтчетовПоУмолчаниюOmnicomm");
	Для НомерСтроки = 1 По МакетНастройки.ВысотаТаблицы Цикл
		
		ИмяПараметра = СокрЛП(МакетНастройки.Область(НомерСтроки, 1).Текст);
		Если СтрНайти(ИмяПараметра, "ИмяГруппыПараметра") <> 0 Тогда
			ИмяПараметра	 = СтрЗаменить(СокрЛП(ИмяПараметра), "ИмяГруппыПараметра_", "");
			Группа			 = ИмяПараметра;
			ЭтоГруппа		 = Истина;
		Иначе
			ИмяПараметра	 = СтрЗаменить(СокрЛП(МакетНастройки.Область(НомерСтроки, 1).Текст), "ИмяПараметра_", "");
		КонецЕсли;
		
		Элемент = Элементы.Найти(ИмяПараметра);
		Если Элемент <> неопределено Тогда
			СтрНаименование   = СокрЛП(МакетНастройки.Область(НомерСтроки, 2).Текст);
			Элемент.Заголовок = СтрНаименование;
		КонецЕсли;
		
	КонецЦикла;
КонецПроцедуры

&НаКлиенте
Процедура movementAndWorkingParamsПриИзменении(Элемент)
	mileage								 = movementAndWorkingParams;
	mileageAverage						 = movementAndWorkingParams;
	mileageSpeeding						 = movementAndWorkingParams;
	speedAverage						 = movementAndWorkingParams;
	speedMax							 = movementAndWorkingParams;
	movementTimeTOtal					 = movementAndWorkingParams;
	movementTime						 = movementAndWorkingParams;
	engineOperationTime					 = movementAndWorkingParams;
	engineOperationTimeInMovement		 = movementAndWorkingParams;
	engineOperationTimeWithoutMovement	 = movementAndWorkingParams;
	engineIdlingTime					 = movementAndWorkingParams;
	engineOperationTimeNormalSpeed		 = movementAndWorkingParams;
	engineOperationTimeMaxSpeed			 = movementAndWorkingParams;
	engineOffTime						 = movementAndWorkingParams;
	odometerInitial						 = movementAndWorkingParams;
	odometerFinal						 = movementAndWorkingParams;
КонецПроцедуры

&НаКлиенте
Процедура fuelParamsПриИзменении(Элемент)
	fuelVolumeInitial				 = fuelParams;
	fuelVolumeFinal					 = fuelParams;
	fuelConsumptionActual			 = fuelParams;
	fuelConsumptionActualAverage	 = fuelParams;
	fuellingsVolume					 = fuelParams;
	fillsVolume						 = fuelParams;
	drainingsVolume					 = fuelParams;
	fuelDispensingVolume			 = fuelParams;
	possibleDrainOrExcess			 = fuelParams;
	fuelVolumeMin					 = fuelParams;
	fuelVolumeMax					 = fuelParams;
	
	fuelConsumptionActualPer100km								 = fuelParams;
	fuelConsumptionActualInMovementPer100km						 = fuelParams;
	fuelConsumptionActualInMovement								 = fuelParams;
	fuelConsumptionActualWithoutMovement						 = fuelParams;
	fuelConsumptionRatedPer100km								 = fuelParams;
	fuelVolumeCalculatedByFuelConsumptionRatedPer100km			 = fuelParams;
	fuelConsumptionDeviationInPercentPer100km					 = fuelParams;
	fuelConsumptionDeviationPer100km							 = fuelParams;
	fuelConsumptionActualPer1HourEngineOperation				 = fuelParams;
	fuelConsumptionActualPer1HourEngineOperationWithoutMovement	 = fuelParams;
	fuelConsumptionRatedPer1HourEngineOperation					 = fuelParams;
	
	fuelConsumptionRatedByPeriod							 = fuelParams;
	fuelConsumptionDeviationInPercentPer1HourEngineOperation = fuelParams;
	fuelConsumptionDeviationPer1HourEngineOperation			 = fuelParams;
	fuelConsumptionActualInMovementIdleEngineSpeed			 = fuelParams;
	fuelConsumptionActualWithoutMovementIdleEngineSpeed		 = fuelParams;
	fuelConsumptionActualInMovementNormalEngineSpeed		 = fuelParams;
	fuelConsumptionActualWithoutMovementNormalEngineSpeed	 = fuelParams;
	fuelConsumptionActualWithoutMovementMaxEngineSpeed		 = fuelParams;
	engineIdleFuelConsumptionDeviationFromRatedPer1Hour		 = fuelParams;
	fuelConsumptionActualInMovementMaxEngineSpeed			 = fuelParams;
КонецПроцедуры

#КонецОбласти